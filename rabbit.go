package rrpc

import (
	"errors"
	"log"
	"runtime"
	"sync"

	//"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

var cpus = runtime.NumCPU()

var ChannelClosed = errors.New("channel closed")

type RabbitDesc struct {
	Url         string
	Queue       string
	Wait        bool
	Connections int
}

type Rabbit struct {
	desc *RabbitDesc
	conn *amqp.Connection
	once *sync.Once

	in  chan *Payload
	out chan *Payload

	wg   *sync.WaitGroup
	stop chan bool
}

// Initialise a new Rabbit
func NewRabbit(rd *RabbitDesc, in chan *Payload, out chan *Payload) (*Rabbit, error) {
	r := &Rabbit{
		desc: rd,
		once: &sync.Once{},
		in:   in,
		out:  out,
		wg:   &sync.WaitGroup{},
		stop: make(chan bool),
	}

	for i := 0; i < r.desc.Connections; i++ {
		log.Println("connecting...")
		if err := r.NewConn(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Rabbit) NewConn() (err error) {
	log.Println("dialing")
	r.once.Do(func() { r.conn, err = amqp.Dial(r.desc.Url) })
	if err != nil {
		return err
	}
	log.Println("dial")

	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}

	log.Println("channel")

	q, err := ch.QueueDeclare(
		r.desc.Queue, // name
		true,         // durable
		false,        // delete when usused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return err
	}

	log.Println("queue")

	if err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return err
	}

	log.Println("qos")

	consume, err := ch.Consume(
		q.Name,       // queue
		"",           // consumer
		!r.desc.Wait, // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return err
	}

	log.Println("consume")

	go func() {
		if err := func() error {
			defer r.wg.Done()
			defer ch.Close()

			for {
				log.Println(r.desc.Queue, "...")
				select {
				case pl, ok := <-r.out:
					if !ok {
						return ChannelClosed
					}
					if err = ch.Publish("", pl.Route, false, false, pl.Publish()); err != nil {
						return err
					}

				case <-r.stop:
					return nil

				case d, ok := <-consume:
					if !ok {
						return ChannelClosed
					}

					r.in <- Deliver(&d)

					if err := d.Ack(false); err != nil {
						return err
					}
				}
			}

		}(); err != nil {
			log.Println("ERR: ", err)
		}
	}()
	return nil
}

func (r *Rabbit) Manage() error {
	return nil
}
