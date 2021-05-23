// Package session abstracts around the REST API and the Gateway, managing both
// at once. It offers a handler interface similar to that in discordgo for
// Gateway events.
package session

import (
	"context"
	"strings"

	"github.com/diamondburned/arikawa/v2/gateway/shard"
	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/internal/handleloop"
	"github.com/diamondburned/arikawa/v2/utils/handler"
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
	ShardManager *shard.Manager

	// Command handler with inherited methods.
	*handler.Handler

	// internal state to not be copied around.
	looper *handleloop.Loop
}

func NewWithIntents(token string, intents ...gateway.Intents) (*Session, error) {
	g, err := gateway.NewGatewayWithIntents(token, intents...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Gateway")
	}

	return NewWithGateways(g), nil
}

// New creates a new session from a given token. Most bots should be using
// NewWithIntents instead.
func New(token string) (*Session, error) {
	if !strings.HasPrefix(token, "Bot") {
		gw, err := gateway.NewGateway(token)
		if err != nil {
			return nil, err
		}

		return NewWithGateways(gw), nil
	}

	m, err := shard.NewManager(token)
	if err != nil {
		return nil, err
	}

	return NewWithShardManager(m), err
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

func NewWithGateways(gw ...*gateway.Gateway) *Session {
	return NewWithShardManager(shard.NewManagerWithGateways(gw...))
}

func NewWithShardManager(m *shard.Manager) *Session {
	handler := handler.New()
	looper := handleloop.NewLoop(handler)

	return &Session{
		ShardManager: m,
		// Nab off gateway's token
		Client:  api.NewClient(m.Gateways()[0].Identifier.Token),
		Handler: handler,
		looper:  looper,
	}
}

func (s *Session) Open() error {
	// Start the handler beforehand so no events are missed.
	s.looper.Start(s.ShardManager.Events)

	// Set the AfterClose's handler.
	s.ShardManager.Apply(func(g *gateway.Gateway) error {
		g.AfterClose = func(err error) {
			s.Handler.Call(&Closed{Error: err})
		}

		return nil
	})

	if err := s.ShardManager.Open(); err != nil {
		return errors.Wrap(err, "failed to start gateway")
	}

	return nil
}

// WithContext returns a shallow copy of Session with the context replaced in
// the API client. All methods called on the returned Session will use this
// given context.
//
// This method is thread-safe only after Open and before Close are called. Open
// and Close should not be called on the returned Session.
func (s *Session) WithContext(ctx context.Context) *Session {
	cpy := *s
	cpy.Client = s.Client.WithContext(ctx)
	return &cpy
}

// Close closes the gateway gracefully.
func (s *Session) Close() error {
	s.looper.Stop()
	return s.ShardManager.Close()
}
