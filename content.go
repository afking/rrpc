package rrpc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

var pbPool = &sync.Pool{
	New: func() interface{} {
		return proto.NewBuffer([]byte{})
	},
}

func Dec(b []byte, pb proto.Message) error {
	buf := pbPool.Get().(*proto.Buffer)
	defer pbPool.Put(buf)
	defer buf.Reset()

	buf.SetBuf(b)
	return buf.Unmarshal(pb)
}

func Enc(pb proto.Message) ([]byte, error) {
	buf := pbPool.Get().(*proto.Buffer)
	defer pbPool.Put(buf)
	defer buf.Reset()

	return buf.Bytes(), buf.Marshal(pb)
}

const (
	TypeServe string = "serve"
	TypeReply string = "reply"
)

// Payload abstracts amqp.Delivery and amqp.Publishing
type Payload struct {
	Route string // ->
	Reply string // <-

	Exp time.Time
	Typ string

	CorId string
	MsgId string
	AppId string

	Body []byte
}

func Conveyor() chan *Payload { return make(chan *Payload, cpus) }

func Deliver(d *amqp.Delivery) *Payload {
	return &Payload{
		Route: "",
		Reply: d.ReplyTo,
		Exp:   d.Timestamp,
		Typ:   d.Type,
		CorId: d.CorrelationId,
		MsgId: d.MessageId,
		AppId: d.AppId,
		Body:  d.Body,
	}
}
func (pl *Payload) Publish() amqp.Publishing {
	return amqp.Publishing{
		//Headers: amqp.Table,

		ContentType:     "application/octet-stream", // TODO: type support
		ContentEncoding: "",
		//DeliveryMode    uint8     // Transient (0 or 1) or Persistent (2)
		//Priority        uint8     // 0 to 9
		CorrelationId: "",
		ReplyTo:       pl.Reply,
		//Expiration: "",
		MessageId: pl.MsgId,
		Timestamp: pl.Exp,
		Type:      pl.Typ,
		//UserId:    "",
		AppId: pl.AppId,

		Body: pl.Body,
	}
}
func (pl *Payload) Context() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(context.Background(), pl.Exp)

	ctx = trace.NewContext(ctx, trace.New(pl.AppId, pl.CorId+"."+pl.MsgId))

	return ctx, cancel
}

func (s *Service) NewPayload(queue, message, typ string, deadline time.Time, b []byte) *Payload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cid string
	for {
		cid = fmt.Sprintf("%010d", atomic.AddUint32(&s.count, 1))
		if _, ok := s.route[cid]; !ok {
			break
		}
	}

	return &Payload{
		Route: queue,
		Reply: s.rabbit.desc.Queue, // SELF: TODO <- FIXME

		Exp: deadline,
		Typ: TypeServe,

		CorId: cid,
		MsgId: message, // <- string type
		AppId: "",      // TODO

		Body: b,
	}
}
