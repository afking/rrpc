package main

import (
	"log"
	"time"

	"github.com/afking/rrpc"
	"github.com/afking/rrpc/example/ping"
	"golang.org/x/net/context"
)

// TODO: Nicer interface methods

var server = rrpc.NewService()

var serverRabbit = &rrpc.RabbitDesc{
	Url:         "amqp://guest:guest@localhost:5672/",
	Queue:       "rabbit-server",
	Wait:        true,
	Connections: 1,
}

type PingServer struct {
	// Implements ping.PingServiceServer
}

func (*PingServer) Ping(ctx context.Context, p *ping.PingRequest) (*ping.PingResponse, error) {
	t := time.Now()

	return &ping.PingResponse{
		Seconds: int64(t.Second()),
		Nanos:   int32(t.Nanosecond()),
	}, nil
}

var client = rrpc.NewService()

var clientRabbit = &rrpc.RabbitDesc{
	Url:         "amqp://guest:guest@localhost:5672/",
	Queue:       "rabbit-client",
	Wait:        true,
	Connections: 1,
}

func main() {
	log.Println("hello")

	// Lets run both client and server

	server.RegisterRabbit(serverRabbit)
	client.RegisterRabbit(clientRabbit)

	ping.RegisterPingServiceServer(server, &PingServer{}) // Ping Server

	// TODO: TAKE QUEUE ARGUMENT ->
	pc := ping.NewPingServiceClient(client) // Ping Client

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*1))

	pong, err := pc.Ping(ctx, &ping.PingRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("%v", time.Unix(pong.Seconds, int64(pong.Nanos)))

	// TODO: Errors...
}
