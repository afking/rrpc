package rrpc

import (
	"log"
	"testing"
	"time"
)

var server = &RabbitDesc{
	Url:         "amqp://guest:guest@localhost:5672/",
	Queue:       "rabbit-server",
	Wait:        true,
	Connections: 1,
}

var client = &RabbitDesc{
	Url:         "amqp://guest:guest@localhost:5672/",
	Queue:       "rabbit-client",
	Wait:        true,
	Connections: 1,
}

func pass(in chan *Payload, out chan *Payload) {
	for pl := range in {
		pl.Route = pl.Reply
		pl.Reply = ""
		out <- pl
	}
}

func TestConns(t *testing.T) {

	// TODO: Improve starting conditions

	s, err := NewRabbit(server, Conveyor(), Conveyor())
	if err != nil {
		t.Fatal(err)
	}

	c, err := NewRabbit(client, Conveyor(), Conveyor())
	if err != nil {
		t.Fatal(err)
	}

	log.Println("started clients")

	go pass(s.in, s.out)

	pl := &Payload{
		Route: s.desc.Queue,
		Reply: c.desc.Queue,
		Body:  []byte(" ／(^ x ^=)＼ "),
	}

	start := time.Now()
	c.out <- pl
	log.Println("sent payload")
	pl = <-c.in
	t.Log(time.Now().Sub(start))
}
