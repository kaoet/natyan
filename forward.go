package main

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/onetwogoo/natyan/proto"
)

func doRemoteForward(server string, channel int32, endpoint string) {
	serverHost, _, err := net.SplitHostPort(server)
	if err != nil {
		log.WithError(err).Fatal("split server")
	}

	conn, err := grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Fatal("connect relay server")
	}
	client := natyan.NewNatyanClient(conn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		resp, err := client.Accept(ctx, &natyan.AcceptRequest{Channel: channel})
		cancel()
		if err != nil {
			if grpc.Code(err) != codes.DeadlineExceeded {
				log.WithError(err).Error("relay accept")
				time.Sleep(10 * time.Second)
			}
			continue
		}

		go dialAndPipe(net.JoinHostPort(serverHost, fmt.Sprint(resp.Port)), endpoint)
	}
}

func doLocalForward(server, endpoint string, channel int32) {
	serverHost, _, err := net.SplitHostPort(server)
	if err != nil {
		log.WithError(err).Fatal("split server")
	}

	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.WithError(err).Fatal("listen")
	}

	conn, err := grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Fatal("connect relay server")
	}
	client := natyan.NewNatyanClient(conn)

	for {
		connA, err := listener.Accept()
		if err != nil {
			log.WithError(err).Fatal("accept")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := client.Connect(ctx, &natyan.ConnectRequest{Channel: channel})
		cancel()
		if err != nil {
			log.WithError(err).Error("relay connect")
			connA.Close()
			return
		}

		connB, err := net.DialTimeout("tcp", net.JoinHostPort(serverHost, fmt.Sprint(resp.Port)), 10*time.Second)
		if err != nil {
			log.WithError(err).WithField("relayPort", resp.Port).Warn("dial")
			connA.Close()
			return
		}

		go func() {
			pipeConn(connA.(*net.TCPConn), connB.(*net.TCPConn))
			connA.Close()
			connB.Close()
		}()
	}
}
