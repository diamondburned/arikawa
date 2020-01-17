package session

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

/*
	TODO:

	and Session's supposed to handle callbacks too kec

	might move all these to Gateway, dunno

	could have a lock on Listen()

	I can actually see people using gateway channels to handle things
	themselves without any callback abstractions, so this is probably the way to go

	welp shit

	rewrite imminent
*/

type Session struct {
	*api.Client
	gateway *gateway.Gateway

	ErrorLog func(err error) // default to log.Println

	// Synchronous controls whether to spawn each event handler in its own
	// goroutine. Default false (meaning goroutines are spawned).
	Synchronous bool

	// handlers stuff
	handlers map[uint64]handler
	hserial  uint64
	hmutex   sync.Mutex
	hstop    chan<- struct{}
}

func New(token string) (*Session, error) {
	// Initialize the session and the API interface
	s := &Session{}
	s.Client = api.NewClient(token)

	// Default logger
	s.ErrorLog = func(err error) {
		log.Println("Arikawa/session error:", err)
	}

	// Open a gateway
	g, err := gateway.NewGateway(token)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Gateway")
	}
	s.gateway = g

	return s, nil
}

func NewWithGateway(gw *gateway.Gateway) *Session {
	return &Session{
		// Nab off gateway's token
		Client: api.NewClient(gw.Identifier.Token),
		ErrorLog: func(err error) {
			log.Println("Arikawa/session error:", err)
		},
		handlers: map[uint64]handler{},
	}
}

func (s *Session) Open() error {
	if err := s.gateway.Start(); err != nil {
		return errors.Wrap(err, "Failed to start gateway")
	}

	stop := make(chan struct{})
	s.hstop = stop
	go s.startHandler(stop)

	return nil
}

func (s *Session) AddHandler(handler interface{}) (rm func()) {
	rm, err := s.addHandler(handler)
	if err != nil {
		panic(err)
	}
	return rm
}

// AddHandlerCheck adds the handler, but safe-guards reflect panics with a
// recoverer, returning the error.
func (s *Session) AddHandlerCheck(handler interface{}) (rm func(), err error) {
	// Reflect would actually panic if anything goes wrong, so this is just in
	// case.
	defer func() {
		if rec := recover(); rec != nil {
			if recErr, ok := rec.(error); ok {
				err = recErr
			} else {
				err = fmt.Errorf("%v", rec)
			}
		}
	}()

	return s.addHandler(handler)
}

func (s *Session) addHandler(handler interface{}) (rm func(), err error) {
	// Reflect the handler
	h, err := reflectFn(handler)
	if err != nil {
		return nil, errors.Wrap(err, "Handler reflect failed")
	}

	s.hmutex.Lock()
	defer s.hmutex.Unlock()

	// Get the current counter value and increment the counter
	serial := s.hserial
	s.hserial++

	// Use the serial for the map
	s.handlers[serial] = *h

	return func() {
		s.hmutex.Lock()
		defer s.hmutex.Unlock()

		delete(s.handlers, serial)
	}, nil
}

func (s *Session) startHandler(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case ev := <-s.gateway.Events:
			s.call(ev)
		}
	}
}

func (s *Session) call(ev interface{}) {
	var evV = reflect.ValueOf(ev)
	var evT = evV.Type()

	s.hmutex.Lock()
	defer s.hmutex.Unlock()

	for _, handler := range s.handlers {
		if handler.not(evT) {
			continue
		}

		if s.Synchronous {
			handler.call(evV)
		} else {
			go handler.call(evV)
		}
	}
}

func (s *Session) Close() error {
	// Stop the event handler
	if s.hstop != nil {
		close(s.hstop)
	}

	// Close the websocket
	return s.gateway.Close()
}
