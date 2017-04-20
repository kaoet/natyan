package main

import (
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/onetwogoo/natyan/proto"
)

const (
	minPort = 32768
	maxPort = 65536
)

type natyanServer struct {
	clientPorts map[int32]chan int
	mu          sync.Mutex
}

func newNatyanServer() *natyanServer {
	return &natyanServer{
		clientPorts: make(map[int32]chan int),
	}
}

func (s *natyanServer) getPorts(channel int32) chan int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ports, ok := s.clientPorts[channel]
	if ok {
		return ports
	} else {
		ports = make(chan int)
		s.clientPorts[channel] = ports
		return ports
	}
}

func (s *natyanServer) Accept(ctx context.Context, in *natyan.AcceptRequest) (*natyan.AcceptResponse, error) {
	log.WithField("channel", in.Channel).Info("accept")

	select {
	case port := <-s.getPorts(in.Channel):
		return &natyan.AcceptResponse{Port: int32(port)}, nil
	case <-ctx.Done():
		return nil, grpc.Errorf(codes.Canceled, "canceled")
	}
}

func (s *natyanServer) Connect(ctx context.Context, in *natyan.ConnectRequest) (*natyan.ConnectResponse, error) {
	port := rand.Intn(maxPort-minPort) + minPort

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.WithError(err).WithField("port", port).Error("failed to listen")
		return nil, grpc.Errorf(codes.Internal, "internal")
	}

	select {
	case s.getPorts(in.Channel) <- port:
	default:
		listener.Close()
		return nil, grpc.Errorf(codes.Unavailable, "no server available")
	}

	go func() {
		defer listener.Close()

		if err := listener.SetDeadline(time.Now().Add(time.Minute)); err != nil {
			log.WithError(err).Error("set listener deadline")
			return
		}

		connA, err := listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Error("accept")
			return
		}
		defer connA.Close()
		connB, err := listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Error("accept")
			return
		}
		defer connB.Close()

		pipeConn(connA, connB)
	}()

	return &natyan.ConnectResponse{Port: int32(port)}, nil
}
