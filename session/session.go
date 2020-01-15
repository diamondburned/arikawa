package session

import (
	"log"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/json"
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
	API         *api.Client
	Gateway     *gateway.Conn
	gatewayOnce sync.Once

	ErrorLog func(err error) // default to log.Println

	// Heartrate is the received duration between heartbeats.
	Heartrate time.Duration

	// LastBeat logs the received heartbeats, with the newest one
	// first.
	LastBeat [2]time.Time

	// Used for Close()
	stoppers []chan<- struct{}
	closers  []func() error
}

func New(token string) (*Session, error) {
	// Initialize the session and the API interface
	s := &Session{}
	s.API = api.NewClient(token)

	// Default logger
	s.ErrorLog = func(err error) {
		log.Println("Arikawa/session error:", err)
	}

	// Connect to the Gateway
	c, err := gateway.NewConn(json.Default{})
	if err != nil {
		return nil, err
	}
	s.Gateway = c

	return s, nil
}

func (s *Session) Close() error {
	for _, stop := range s.stoppers {
		close(stop)
	}

	var err error

	for _, closer := range s.closers {
		if cerr := closer(); cerr != nil {
			err = cerr
		}
	}

	return err
}
