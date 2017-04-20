package main

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/onetwogoo/natyan/proto"
)

func main() {
	app := kingpin.New("natyan", "Natyan for bypassing NAT")
	app.Version("0.0.1")

	{
		relay := app.Command("relay", "As relay server")
		bind := relay.Flag("bind", "Bind address").Default(":1842").String()
		relay.Action(func(*kingpin.ParseContext) error {
			listener, err := net.Listen("tcp", *bind)
			if err != nil {
				log.WithError(err).Fatal("listen")
			}
			s := grpc.NewServer()
			natyan.RegisterNatyanServer(s, newNatyanServer())
			log.Fatal(s.Serve(listener))
			return nil
		})
	}

	{
		remoteForward := app.Command("remote-forward", "Forward a channel to an endpoint")
		server := remoteForward.Arg("server", "Endpoint of relay server").Required().String()
		channel := remoteForward.Arg("channel", "Channel").Required().Int32()
		endpoint := remoteForward.Arg("endpoint", "Endpoint to receive connections").Required().String()
		remoteForward.Action(func(*kingpin.ParseContext) error {
			doRemoteForward(*server, *channel, *endpoint)
			return nil
		})
	}

	{
		localForward := app.Command("local-forward", "Forward an endpoint to a channel")
		server := localForward.Arg("server", "Endpoint of relay server").Required().String()
		endpoint := localForward.Arg("endpoint", "Endpoint to receive connections").Required().String()
		channel := localForward.Arg("channel", "Channel").Required().Int32()
		localForward.Action(func(*kingpin.ParseContext) error {
			doLocalForward(*server, *endpoint, *channel)
			return nil
		})
	}

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
