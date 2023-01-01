// Package session abstracts around the REST API and the Gateway, managing both
// at once. It offers a handler interface similar to that in discordgo for
// Gateway events.
package session

import (
	"context"
	"log"
	"sync"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/arikawa/v3/utils/ws/ophandler"
)

// ErrMFA is returned if the account requires a 2FA code to log in.
var ErrMFA = errors.New("account has 2FA enabled")

// ErrClosed is returned if the Session is closed, either because it's already
// closed (and Close is being called again) or it was never started.
var ErrClosed = errors.New("Session is closed")

// Session manages both the API and Gateway. As such, Session inherits all of
// API's methods, as well has the Handler used for Gateway.
type Session struct {
	*api.Client
	*handler.Handler

	// internal state to not be copied around.
	state *sessionState

	// OnInteractionError is called when an interaction added using
	// AddInteractionHandler cannot be sent. By default, it logs into the
	// console.
	OnInteractionError func(*gateway.InteractionCreateEvent, error)

	// DontWaitForReady makes Open not wait for the Ready event. This is useful
	// for non-bots, since Discord may send over a READY_SUPPLEMENT instead. If
	// this is true, then any event sent by Discord will unblock Open (usually
	// HELLO).
	DontWaitForReady bool // false
}

type sessionState struct {
	sync.Mutex
	id      gateway.Identifier
	gateway *gateway.Gateway

	ctx    context.Context
	cancel context.CancelFunc
	doneCh <-chan struct{}
}

// NewWithIntents is similar to New but adds the given intents in during
// construction.
func NewWithIntents(token string, intents ...gateway.Intents) *Session {
	var allIntent gateway.Intents
	for _, intent := range intents {
		allIntent |= intent
	}

	id := gateway.DefaultIdentifier(token)
	id.Intents = option.NewUint(uint(allIntent))

	return NewWithIdentifier(id)
}

// New creates a new session from a given token. Most bots should be using
// NewWithIntents instead.
func New(token string) *Session {
	return NewWithIdentifier(gateway.DefaultIdentifier(token))
}

// Login tries to log in as a normal user account; MFA is optional.
func Login(ctx context.Context, email, password, mfa string) (*Session, error) {
	// Make a scratch HTTP client without a token
	client := api.NewClient("").WithContext(ctx)

	// Try to login without TOTP
	l, err := client.Login(email, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to login")
	}

	if l.Token != "" && !l.MFA {
		// We got the token, return with a new Session.
		return New(l.Token), nil
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

	return New(l.Token), nil
}

// NewWithIdentifier creates a bare Session with the given identifier.
func NewWithIdentifier(id gateway.Identifier) *Session {
	return NewCustom(id, api.NewClient(id.Token), handler.New())
}

// NewWithGateway constructs a bare Session from the given UNOPENED gateway.
func NewWithGateway(g *gateway.Gateway, h *handler.Handler) *Session {
	state := g.State()
	client := api.NewClient(state.Identifier.Token)
	return newCustom(state.Identifier, client, h, g)
}

// NewCustom constructs a bare Session from the given parameters.
func NewCustom(id gateway.Identifier, cl *api.Client, h *handler.Handler) *Session {
	return newCustom(id, cl, h, nil)
}

func newCustom(
	id gateway.Identifier,
	cl *api.Client,
	h *handler.Handler,
	g *gateway.Gateway) *Session {

	return &Session{
		Client:  cl,
		Handler: h,
		state: &sessionState{
			gateway: g,
			id:      id,
		},
		OnInteractionError: func(ev *gateway.InteractionCreateEvent, err error) {
			// Log the error by default.
			// TODO: fix this once we resolve
			// https://github.com/diamondburned/arikawa/issues/361.
			log.Printf("session: error handling interaction %v: %v", ev.ID, err)
		},
	}
}

// AddIntents adds the given intents into the gateway. Calling it after Open has
// already been called will result in a panic.
func (s *Session) AddIntents(intents gateway.Intents) {
	s.state.Lock()

	s.state.id.AddIntents(intents)

	if s.state.gateway != nil {
		s.state.gateway.AddIntents(intents)
	}

	s.state.Unlock()
}

// HasIntents reports if the Gateway has the passed Intents.
//
// If no intents are set, e.g. if using a user account, HasIntents will always
// return true.
func (s *Session) HasIntents(intents gateway.Intents) bool {
	return s.state.id.HasIntents(intents)
}

// Gateway returns the current session's gateway. If Open has never been called
// or Session was never constructed with a gateway, then nil is returned.
func (s *Session) Gateway() *gateway.Gateway {
	s.state.Lock()
	defer s.state.Unlock()

	return s.state.gateway
}

// GatewayOpts returns a copy of the current session's gateway options. If Open
// has never been called or Session was never constructed with a gateway, then
// the default gateway options are returned.
func (s *Session) GatewayOpts() *ws.GatewayOpts {
	s.state.Lock()
	defer s.state.Unlock()

	opts := &gateway.DefaultGatewayOpts
	if s.state.gateway != nil {
		opts = s.state.gateway.Opts()
	}

	return opts
}

// GatewayError returns the gateway's error if the gateway is dead. If it's not
// dead, then nil is always returned. The check is done with GatewayIsAlive().
// If the gateway has never been started, nil will be returned (even though
// GatewayIsAlive would've returned true).
//
// This method would return what Close() would've returned if a fatal gateway
// error was found.
func (s *Session) GatewayError() error {
	s.state.Lock()
	defer s.state.Unlock()

	if !s.gatewayIsAlive() && s.state.gateway != nil {
		return s.state.gateway.LastError()
	}

	return nil
}

// GatewayIsAlive returns true if the gateway is still alive, that is, it is
// either connected or is trying to reconnect after an interruption. In other
// words, false is returned if the gateway isn't open or it has exited after
// seeing a fatal error code (and therefore cannot recover).
func (s *Session) GatewayIsAlive() bool {
	s.state.Lock()
	defer s.state.Unlock()

	return s.gatewayIsAlive()
}

func (s *Session) gatewayIsAlive() bool {
	if s.state.gateway == nil || s.state.doneCh == nil {
		return false
	}

	select {
	case <-s.state.doneCh:
		return false
	default:
		return true
	}
}

// Connect opens the Discord gateway and waits until an unrecoverable error
// occurs. Always prefer this method over Open. Note that Connect will return
// when ctx is done or when s.Close is called.
//
// As an odd case, when ctx is done and if the gateway is already finished
// connecting, then a nil error will be returned (unless the gateway has an
// error). This is contrary to the common behavior of a ctx function returning
// ctx.Err().
func (s *Session) Connect(ctx context.Context) error {
	opts := s.GatewayOpts()

	for {
		if err := s.Open(ctx); err != nil {
			if opts.ErrorIsFatalClose(err) || ctx.Err() != nil {
				// Fatal error or context is done, return.
				return err
			}
			// Non-fatal error, retry.
			continue
		}

		if err := s.Wait(ctx); err != nil {
			if opts.ErrorIsFatalClose(err) {
				// Gateway returned a fatal error, so we can't recover.
				return err
			}
			if ctx.Err() != nil {
				// Context was done, so we can't recover. Exit with no error,
				// since we're just waiting.
				return nil
			}
			// Non-fatal error, retry.
		}
	}
}

// Open opens the Discord gateway and its handler, then waits until either the
// Ready or Resumed event gets through. Prefer using Connect instead of Open.
func (s *Session) Open(ctx context.Context) error {
	evCh := make(chan interface{})

	s.state.Lock()
	defer s.state.Unlock()

	if s.state.cancel != nil {
		if err := s.close(); err != nil {
			return err
		}
	}

	if s.state.gateway == nil {
		g, err := gateway.NewWithIdentifier(ctx, s.state.id)
		if err != nil {
			return err
		}
		s.state.gateway = g
	}

	// Make a context that's stored in state so this can be used throughout.
	s.state.ctx, s.state.cancel = context.WithCancel(context.Background())

	// TODO: change this to AddSyncHandler.
	rm := s.AddHandler(evCh)
	defer rm()

	opCh := s.state.gateway.Connect(s.state.ctx)
	s.state.doneCh = ophandler.Loop(opCh, s.Handler)

	for {
		select {
		case <-ctx.Done():
			s.close()
			return ctx.Err()

		case <-s.state.ctx.Done():
			s.close()
			return s.state.ctx.Err()

		case <-s.state.doneCh:
			// Event loop died.
			return s.state.gateway.LastError()

		case ev := <-evCh:
			if s.DontWaitForReady {
				return nil
			}

			switch ev.(type) {
			case *gateway.ReadyEvent, *gateway.ResumedEvent:
				return nil
			}
		}
	}
}

// Wait blocks until either ctx is done or the gateway stumbles on an
// unrecoverable error.
func (s *Session) Wait(ctx context.Context) error {
	s.state.Lock()
	doneCh := s.state.doneCh
	s.state.Unlock()

	if doneCh == nil {
		return ErrClosed
	}

	for {
		select {
		case <-ctx.Done():
			s.Close()
			// Prefer gateway errors over context errors.
			if err := s.GatewayError(); err != nil {
				return err
			}
			return ctx.Err()

		case <-doneCh:
			// Event loop died.
			return s.GatewayError()
		}
	}
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

// AddInteractionHandler adds an interaction handler function to be handled with
// the gateway and the API client. Use this as a compatibility layer for bots
// that support both methods of hosting.
//
// AddInteractionHandler will automatically send the return value of the
// interaction handler to the API. If the return value cannot be sent
// successfully, then s.OnInteractionError will be called.
func (s *Session) AddInteractionHandler(h webhook.InteractionHandler) {
	// State doesn't override this, but it doesn't touch
	// InteractionCreateEvents, so it shouldn't need to.
	s.AddHandler(func(ev *gateway.InteractionCreateEvent) {
		if resp := h.HandleInteraction(&ev.InteractionEvent); resp != nil {
			if err := s.RespondInteraction(ev.ID, ev.Token, *resp); err != nil {
				s.OnInteractionError(ev, err)
			}
		}
	})
}

// AddInteractionHandlerFunc is a function variant of AddInteractionHandler.
func (s *Session) AddInteractionHandlerFunc(f webhook.InteractionHandlerFunc) {
	s.AddInteractionHandler(f)
}

// SendGateway is a helper to send messages over the gateway. It will check
// if the gateway is open and available, then send the message.
func (s *Session) SendGateway(ctx context.Context, m ws.Event) error {
	// The only necessary check here is checking if gateway is nil, however
	// this will save us a bit of work in serialization.
	if !s.GatewayIsAlive() {
		return ErrClosed
	}

	return s.Gateway().Send(ctx, m)
}

// Close closes the underlying Websocket connection, invalidating the session
// ID. It will send a closing frame before ending the connection, closing it
// gracefully. This will cause the bot to appear as offline instantly. To
// prevent this behavior, change Gateway.AlwaysCloseGracefully.
func (s *Session) Close() error {
	s.state.Lock()
	defer s.state.Unlock()

	return s.close()
}

func (s *Session) close() error {
	if s.state.cancel == nil {
		return ErrClosed
	}

	s.state.cancel()
	s.state.cancel = nil
	s.state.ctx = nil

	<-s.state.doneCh
	s.state.doneCh = nil

	return s.state.gateway.LastError()
}
