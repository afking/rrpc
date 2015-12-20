package rrpc

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	//"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

type methodHandler func(srv interface{}, ctx context.Context, b []byte) ([]byte, error)

type MethodDesc struct {
	MethodName string
	Handler    methodHandler
}

type ServiceDesc struct {
	ServiceName string
	HandlerType interface{}
	Methods     []MethodDesc
}

type Method struct {
	handler methodHandler
	service interface{}
}

// Versioning
// <major>.<minor>.<commit#>-<git SHA>-<date>.<time>

type Service struct {
	mu *sync.RWMutex

	/*Id          string    // Unique service id
	Name        string    // Service name
	Description string    // Description of service
	Version     string    // Program version
	Compiled    time.Time // Compiled date*/

	methods map[string]*Method
	//rabbits map[string]*Rabbit
	rabbit *Rabbit // TODO: multi rabbits

	route map[string]chan *Payload
	count uint32

	in   chan *Payload
	out  chan *Payload
	stop chan bool
}

func NewService() *Service {
	return &Service{
		mu: &sync.RWMutex{},

		methods: make(map[string]*Method),
		//rabbits: make(map[string]*Rabbit),

		route: make(map[string]chan *Payload),

		in:   Conveyor(),
		out:  Conveyor(),
		stop: make(chan bool),
	}
}

func timeAtCompile() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func (s *Service) RegisterService(sd *ServiceDesc, srv interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: srv reflection

	for i := range sd.Methods {
		md := &sd.Methods[i]

		if _, ok := s.methods[md.MethodName]; ok {
			log.Fatalf("duplicate method %s registered", md.MethodName)
		}

		s.methods[md.MethodName] = &Method{md.Handler, srv}
	}

}

func (s *Service) RegisterRabbit(rd *RabbitDesc) {
	//s.mu.Lock()
	//defer s.mu.Unlock()

	/*if _, ok := s.rabbits[rd.Queue]; ok {
		log.Fatalf("duplicate queue %s registered", rd.Queue)
	}*/
	var err error
	s.rabbit, err = NewRabbit(rd, s.in, s.out)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Service) Recieve(ctx context.Context) (*Payload, bool) {
	select {
	case b, ok := <-s.in:
		return b, ok
	case <-ctx.Done():
		return nil, false
	}
}

func (s *Service) Send(ctx context.Context, pl *Payload) bool {
	select {
	case s.out <- pl:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Service) Order(ctx context.Context, pl *Payload) (*Payload, bool) {
	reply := s.Handler(pl.CorId)

	if ok := s.Send(ctx, pl); !ok {
		return nil, false
	}

	select {
	case pl, ok := <-reply:
		return pl, ok
	case <-ctx.Done():
		// TODO: Delete route
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.route, pl.CorId)

		return nil, false
	}
}

func (s *Service) parse(pl *Payload) error {
	ctx, cancel := pl.Context()
	defer cancel()

	switch pl.Typ {
	case TypeServe:

		md, ok := s.methods[pl.MsgId]
		if !ok {
			return nil // handle
		}

		b, err := md.handler(md.service, ctx, pl.Body)
		if err != nil {
			return nil // handle
		}

		if pl.Reply == "" {
			break // drop
		}

		pl.Route = pl.Reply
		pl.Reply = ""
		pl.Typ = TypeReply
		pl.Body = b

		if ok := s.Send(ctx, pl); !ok {
			return nil // handle
		}

	case TypeReply:

		if err := s.Route(ctx, pl); err != nil {
			return nil // handle
		}

	default:
		// handle
	}
	return nil
}

func (s *Service) worker() {
	for pl := range s.in {

		select {
		case <-s.stop:
			// s.out <- pump message out
			return
		default:
			if err := s.parse(pl); err != nil {
				log.Println(err) // handle
			}
		}

	}
}

// Invoke is called by generated rrpc code.
func (s *Service) Invoke(ctx context.Context, queue, message string, in, out proto.Message) error {

	b, err := Enc(in)
	if err != nil {
		return err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return context.DeadlineExceeded
	}

	pl, ok := s.Order(ctx, s.NewPayload(queue, message, TypeServe, deadline, b))
	if !ok {
		return nil // something
	}

	return Dec(pl.Body, out)
}

func (s *Service) Listen() error {
	return nil
}

func (s *Service) Close() {
	// Kill the service, drain gracefully

	// kill all the workers
}
