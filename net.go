package main

import (
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func pipeConn(connA, connB *net.TCPConn) {
	if err := connA.SetKeepAlivePeriod(time.Minute); err != nil {
		log.WithError(err).Warn("set keepalive period")
	}
	if err := connA.SetKeepAlive(true); err != nil {
		log.WithError(err).Warn("set keepalive")
	}
	if err := connB.SetKeepAlivePeriod(time.Minute); err != nil {
		log.WithError(err).Warn("set keepalive period")
	}
	if err := connB.SetKeepAlive(true); err != nil {
		log.WithError(err).Warn("set keepalive")
	}

	var wg sync.WaitGroup
	wg.Add(2)
	pipe := func(connA, connB *net.TCPConn) {
		io.Copy(connA, connB)
		connA.CloseRead()
		connB.CloseWrite()
		wg.Done()
	}

	go pipe(connA, connB)
	go pipe(connB, connA)
	wg.Wait()
}

func dialAndPipe(endpointA, endpointB string) {
	connA, err := net.DialTimeout("tcp", endpointA, 10*time.Second)
	if err != nil {
		log.WithError(err).WithField("endpoint", endpointA).Warn("dial")
		return
	}
	defer connA.Close()

	connB, err := net.DialTimeout("tcp", endpointB, 10*time.Second)
	if err != nil {
		log.WithError(err).WithField("endpoint", endpointB).Warn("dial")
		return
	}
	defer connB.Close()

	pipeConn(connA.(*net.TCPConn), connB.(*net.TCPConn))
}
