package session

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/json"
	"github.com/pkg/errors"
)

var Identity = gateway.IdentifyProperties{
	OS:      runtime.GOOS,
	Browser: "Arikawa",
	Device:  "Arikawa",
}

type Session struct {
	API         api.Client
	Gateway     gateway.Conn
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
	s.API = *api.NewClient(token)

	// Default logger
	s.ErrorLog = func(err error) {
		log.Println("Arikawa/session error:", err)
	}

	// Connect to the Gateway
	c, err := gateway.NewConn(json.Default{})
	if err != nil {
		return nil, err
	}
	s.Gateway = *c

	if err := s.StartGateway(); err != nil {
		return nil, errors.Wrap(err, "Failed to start gateway")
	}

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

// StartGateway is called by New and should only be called once. This method is
// guarded with a sync.Do.
func (s *Session) StartGateway() error {
	var err error
	s.gatewayOnce.Do(func() {
		err = s.startGateway()
	})
	return err
}

// Reconnects and resumes.
func (s *Session) Reconnect() error {
	panic("TODO")
}

func (s *Session) startGateway() error {
	// This is where we'll get our events
	ch := s.Gateway.Listen()

	// Wait for an OP 10 Hello
	var hello gateway.HelloEvent
	if err := gateway.AssertEvent(
		s.Gateway.JSON, <-ch, gateway.HeartbeatOP, &hello); err != nil {

		return errors.Wrap(err, "Error at Hello")
	}

	// Start the pacemaker with the heartrate received from Hello
	s.Heartrate = hello.HeartbeatInterval.Duration()
	go s.startPacemaker()

	return nil
}

func (s *Session) startListener() {
	ch := s.Gateway.Listen()
	stop := s.stopper()

	for {
		select {
		case <-stop:
			return
		case v, ok := <-ch:
			if !ok {
				return
			}

			op, err := gateway.DecodeOP(s.Gateway.JSON, v)
			if err != nil {
				s.ErrorLog(errors.Wrap(err, "Failed to decode OP in loop"))
			}

			if err := s.handleOP(op); err != nil {
				s.ErrorLog(err)
			}
		}
	}
}

func (s *Session) handleOP(op *gateway.OP) error {
	switch op.Code {
	case gateway.HeartbeatAckOP:
		// Swap our received heartbeats
		s.LastBeat[0], s.LastBeat[1] = time.Now(), s.LastBeat[0]
	}

	return nil
}

func (s *Session) startPacemaker() {
	stop := s.stopper()
	tick := time.NewTicker(s.Heartrate)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			if err := s.Gateway.Heartbeat(); err != nil {
				s.ErrorLog(errors.Wrap(err, "Failed to send heartbeat"))
			}

			// Check and see if heartbeats have timed out.
			// TODO: better way?
			if s.LastBeat[0].Sub(s.LastBeat[1]) > s.Heartrate {

				if err := s.Reconnect(); err != nil {
					s.ErrorLog(errors.Wrap(err,
						"Failed to reconnect after heartrate timeout"))
				}
			}
		}
	}
}

func (s *Session) stopper() <-chan struct{} {
	stop := make(chan struct{})
	s.stoppers = append(s.stoppers, stop)
	return stop
}
