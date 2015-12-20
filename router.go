package rrpc

import (
	"errors"

	"golang.org/x/net/context"
)

var UnkownRoute = errors.New("unknown route")

/*type Router interface {
	//Correlate(pl *Payload) // move to something else
	Handle(pl *Payload) chan *Payload
	Route(ctx context.Context, pl *Payload) error
}

type router struct {
	*sync.Mutex
	route map[string]chan *Payload
	count uint32
}
*/

func (s *Service) Handler(cid string) chan *Payload {
	s.mu.Lock()
	defer s.mu.Unlock()

	/*if p.CorrelationId != "" {
		if reply, ok := r.route[cid]; ok {
			return reply
		}
	}*/

	reply := make(chan *Payload)
	s.route[cid] = reply
	return reply
}

func (s *Service) Route(ctx context.Context, pl *Payload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reply, ok := s.route[pl.CorId]
	if !ok {
		return UnkownRoute
	}
	delete(s.route, pl.CorId)

	select {
	case reply <- pl:
		return nil
	case <-ctx.Done():
		return context.DeadlineExceeded
	}
}
