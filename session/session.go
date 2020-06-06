// Package session abstracts around the REST API and the Gateway, managing both
// at once. It offers a handler interface similar to that in discordgo for
// Gateway events.
package session

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
	"github.com/diamondburned/arikawa/utils/moreatomic"

	"github.com/pkg/errors"
)

var ErrMFA = errors.New("account has 2FA enabled")

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

	// unavailableGuilds is a set of discord.Snowflakes of guilds that became
	// unavailable when already connected to the gateway, i.e. sent in a
	// GuildUnavailableEvent.
	unavailableGuilds *moreatomic.SnowflakeSet
	// unreadyGuilds is a set of discord.Snowflakes of guilds that were
	// unavailable when connecting to the gateway, i.e. they had Unavailable
	// set to true during Ready.
	unreadyGuilds *moreatomic.SnowflakeSet
	// guildTrackMutex is the mutex that secures the two sets keeping track
	// of unavailable guilds.
}

func New(token string) (*Session, error) {
	// Create a gateway
	g, err := gateway.NewGateway(token)
	if err != nil {
		err = errors.Wrap(err, "failed to connect to Gateway")
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
		Client:            api.NewClient(gw.Identifier.Token),
		Handler:           handler.New(),
		unavailableGuilds: moreatomic.NewSnowflakeSet(),
		unreadyGuilds:     moreatomic.NewSnowflakeSet(),
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
			s.handleEvent(ev)
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
		s.hstop <- struct{}{}
	}
}
