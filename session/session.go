// Package session abstracts around the REST API and the Gateway, managing both
// at once. It offers a handler interface similar to that in discordgo for
// Gateway events.
package session

import (
	"log"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
	"github.com/pkg/errors"
)

// Session manages both the API and Gateway. As such, Session inherits all of
// API's methods, as well has the Handler used for Gateway.
type Session struct {
	*api.Client
	gateway *gateway.Gateway

	// ErrorLog logs errors, including Gateway errors.
	ErrorLog func(err error) // default to log.Println

	// Command handler with inherited methods.
	*handler.Handler

	hstop chan struct{}
}

func New(token string) (*Session, error) {
	// Initialize the session and the API interface
	s := &Session{}
	s.Handler = handler.New()
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
	s.gateway.ErrorLog = func(err error) {
		s.ErrorLog(err)
	}

	return s, nil
}

func NewWithGateway(gw *gateway.Gateway) *Session {
	s := &Session{
		// Nab off gateway's token
		Client: api.NewClient(gw.Identifier.Token),
		ErrorLog: func(err error) {
			log.Println("Arikawa/session error:", err)
		},
		Handler: handler.New(),
	}

	gw.ErrorLog = func(err error) {
		s.ErrorLog(err)
	}

	return s
}

func (s *Session) Open() error {
	if err := s.gateway.Open(); err != nil {
		return errors.Wrap(err, "Failed to start gateway")
	}

	stop := make(chan struct{})
	s.hstop = stop
	go s.startHandler(stop)

	return nil
}

func (s *Session) startHandler(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case ev := <-s.gateway.Events:
			s.Handler.Call(ev)
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
