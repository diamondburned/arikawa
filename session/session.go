// Package session abstracts around the REST API and the Gateway, managing both
// at once. It offers a handler interface similar to that in discordgo for
// Gateway events.
package session

import (
	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
)

var ErrMFA = errors.New("account has 2FA enabled")

// Closed is an event that's sent to Session's command handler. This works by
// using (*Gateway).AfterClose. If the user sets this callback, no Closed events
// would be sent.
//
// Usage
//
//    ses.AddHandler(func(*session.Closed) {})
//
type Closed struct {
	Error error
}

// Session manages both the API and Gateway. As such, Session inherits all of
// API's methods, as well has the Handler used for Gateway.
type Session struct {
	*api.Client
	Gateway *gateway.Gateway

	// Command handler with inherited methods.
	*handler.Handler

	// MFA only fields
	MFA    bool
	Ticket string

	hstop chan struct{}
}

func New(token string) (*Session, error) {
	// Create a gateway
	g, err := gateway.NewGateway(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Gateway")
	}

	return NewWithGateway(g), err
}

// Login tries to log in as a normal user account; MFA is optional.
func Login(email, password, mfa string) (*Session, error) {
	// Make a scratch HTTP client without a token
	client := api.NewClient("")

	// Try to login without TOTP
	l, err := client.Login(email, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to login")
	}

	if l.Token != "" && !l.MFA {
		// We got the token, return with a new Session.
		return New(l.Token)
	}

	// Discord requests MFA, so we need the MFA token.
	if mfa == "" {
		return nil, ErrMFA
	}

	// Retry logging in with a 2FA token
	l, err = client.TOTP(mfa, l.Ticket)
	if err != nil {
		return nil, errors.Wrap(err, "failed to login with 2FA")
	}

	return New(l.Token)
}

func NewWithGateway(gw *gateway.Gateway) *Session {
	return &Session{
		Gateway: gw,
		// Nab off gateway's token
		Client:  api.NewClient(gw.Identifier.Token),
		Handler: handler.New(),
	}
}

func (s *Session) Open() error {
	// Start the handler beforehand so no events are missed.
	stop := make(chan struct{})
	s.hstop = stop
	go s.startHandler(stop)

	// Set the AfterClose's handler.
	s.Gateway.AfterClose = func(err error) {
		s.Handler.Call(&Closed{
			Error: err,
		})
	}

	if err := s.Gateway.Open(); err != nil {
		return errors.Wrap(err, "failed to start gateway")
	}

	return nil
}

func (s *Session) startHandler(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case ev := <-s.Gateway.Events:
			s.Call(ev)
		}
	}
}

func (s *Session) Close() error {
	// Stop the event handler
	s.close()

	// Close the websocket
	return s.Gateway.Close()
}

func (s *Session) close() {
	if s.hstop != nil {
		close(s.hstop)
	}
}
