// Package gateway handles the Discord gateway (or Websocket) connection, its
// events, and everything related to it. This includes logging into the
// Websocket.
//
// This package does not abstract events and function handlers; instead, it
// leaves that to the session package. This package exposes only a single Events
// channel.
package gateway

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/pkg/errors"
)

var (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = api.Version
	Encoding = "json"
)

var (
	ErrMissingForResume = errors.New("missing session ID or sequence for resuming")
	ErrWSMaxTries       = errors.New(
		"could not connect to the Discord gateway before reaching the timeout")
)

// BotData contains the GatewayURL as well as extra metadata on how to
// shard bots.
type BotData struct {
	URL        string             `json:"url"`
	Shards     int                `json:"shards,omitempty"`
	StartLimit *SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit is the information on the current session start limit. It's
// used in BotData.
type SessionStartLimit struct {
	Total      int                  `json:"total"`
	Remaining  int                  `json:"remaining"`
	ResetAfter discord.Milliseconds `json:"reset_after"`
}

// URL asks Discord for a Websocket URL to the Gateway.
func URL() (string, error) {
	var g BotData

	return g.URL, httputil.NewClient().RequestJSON(
		&g, "GET",
		EndpointGateway,
	)
}

// BotURL fetches the Gateway URL along with extra metadata. The token
// passed in will NOT be prefixed with Bot.
func BotURL(token string) (*BotData, error) {
	var g *BotData

	return g, httputil.NewClient().RequestJSON(
		&g, "GET",
		EndpointGatewayBot,
		httputil.WithHeaders(http.Header{
			"Authorization": {token},
		}),
	)
}

type Gateway struct {
	WS *wsutil.Websocket

	// WSTimeout is a timeout for an arbitrary action. An example of this is the
	// timeout for Start and the timeout for sending each Gateway command
	// independently.
	WSTimeout time.Duration
	// ReconnectTimeout is the timeout used during reconnection.
	// If the a connection to the gateway can't be established before the
	// duration passes, the Gateway will be closed and FatalErrorCallback will
	// be called.
	//
	// Setting this to 0 is equivalent to no timeout.
	ReconnectTimeout time.Duration

	// All events sent over are pointers to Event structs (structs suffixed with
	// "Event"). This shouldn't be accessed if the Gateway is created with a
	// Session.
	Events chan Event

	sessionMu sync.RWMutex
	sessionID string

	Identifier *Identifier
	Sequence   *moreatomic.Int64

	PacerLoop wsutil.PacemakerLoop

	ErrorLog func(err error) // default to log.Println
	// FatalErrorCallback is called, if the Gateway exits fatally. At the point
	// of calling, the gateway will be already closed.
	//
	// Currently this will only be called, if the ReconnectTimeout was changed
	// to a definite timeout, and connection could not be established during
	// that time.
	// err will be ErrWSMaxTries in that case.
	//
	// Defaults to noop.
	FatalErrorCallback func(err error)

	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions.
	AfterClose func(err error) // noop by default

	waitGroup sync.WaitGroup
}

// NewGatewayWithIntents creates a new Gateway with the given intents and the
// default stdlib JSON driver. Refer to NewGatewayWithDriver and AddIntents.
func NewGatewayWithIntents(token string, intents ...Intents) (*Gateway, error) {
	g, err := NewGateway(token)
	if err != nil {
		return nil, err
	}

	for _, intent := range intents {
		g.AddIntents(intent)
	}

	return g, nil
}

// NewGateway creates a new Gateway with the default stdlib JSON driver. For
// more information, refer to NewGatewayWithDriver.
func NewGateway(token string) (*Gateway, error) {
	URL, err := URL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gateway endpoint")
	}

	// Parameters for the gateway
	param := url.Values{
		"v":        {Version},
		"encoding": {Encoding},
	}

	// Append the form to the URL
	URL += "?" + param.Encode()

	return NewCustomGateway(URL, token), nil
}

func NewCustomGateway(gatewayURL, token string) *Gateway {
	return &Gateway{
		WS:        wsutil.NewCustom(wsutil.NewConn(), gatewayURL),
		WSTimeout: wsutil.WSTimeout,

		Events:     make(chan Event, wsutil.WSBuffer),
		Identifier: DefaultIdentifier(token),
		Sequence:   moreatomic.NewInt64(0),

		ErrorLog:   wsutil.WSError,
		AfterClose: func(error) {},
	}
}

// AddIntents adds a Gateway Intent before connecting to the Gateway. As such,
// this function will only work before Open() is called.
func (g *Gateway) AddIntents(i Intents) {
	g.Identifier.Intents |= i
}

// HasIntents reports if the Gateway has the passed Intents.
//
// If no intents are set, i.e. if using a user account HasIntents will always
// return true.
func (g *Gateway) HasIntents(intents Intents) bool {
	if g.Identifier.Intents == 0 {
		return true
	}

	return g.Identifier.Intents.Has(intents)
}

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() error {
	return g.close(false)
}

// CloseGracefully attempts to close the gateway connection gracefully, by
// sending a closing frame before ending the connection. This will cause the
// gateway's session id to be rendered invalid.
//
// Note that a graceful closure is only possible, if the wsutil.Connection of
// the Gateway's Websocket implements wsutil.GracefulCloser.
func (g *Gateway) CloseGracefully() error {
	return g.close(true)
}

func (g *Gateway) close(graceful bool) (err error) {
	wsutil.WSDebug("Trying to close. Pacemaker check skipped.")
	wsutil.WSDebug("Closing the Websocket...")

	if graceful {
		err = g.WS.CloseGracefully()
	} else {
		err = g.WS.Close()
	}

	if errors.Is(err, wsutil.ErrWebsocketClosed) {
		wsutil.WSDebug("Websocket already closed.")
		return nil
	}

	// Explicitly signal the pacemaker loop to stop. We should do this in case
	// the Start function exited before it could bind the event channel into the
	// loop.
	g.PacerLoop.Stop()
	wsutil.WSDebug("Websocket closed; error:", err)

	wsutil.WSDebug("Waiting for the Pacemaker loop to exit.")
	g.waitGroup.Wait()
	wsutil.WSDebug("Pacemaker loop exited.")

	g.AfterClose(err)
	wsutil.WSDebug("AfterClose callback finished.")

	return err
}

// SessionID returns the session ID received after Ready. This function is
// concurrently safe.
func (g *Gateway) SessionID() string {
	g.sessionMu.RLock()
	defer g.sessionMu.RUnlock()

	return g.sessionID
}

// Reconnect tries to reconnect until the ReconnectTimeout is reached, or if
// set to 0 reconnects indefinitely.
func (g *Gateway) Reconnect() {
	ctx := context.Background()

	if g.ReconnectTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(context.Background(), g.ReconnectTimeout)

		defer cancel()
	}

	// ignore the error, it is already logged and FatalErrorCallback was called
	g.ReconnectCtx(ctx)
}

// ReconnectCtx attempts to reconnect until context expires.
// If the context expires FatalErrorCallback will be called with ErrWSMaxTries,
// and the last error returned by Open will be returned.
func (g *Gateway) ReconnectCtx(ctx context.Context) (err error) {
	wsutil.WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	g.Close()

	for try := 1; ; try++ {
		select {
		case <-ctx.Done():
			g.FatalErrorCallback(ErrWSMaxTries)
			return err
		default:
		}

		wsutil.WSDebug("Trying to dial, attempt", try)

		if oerr := g.OpenContext(ctx); oerr != nil {
			g.ErrorLog(err)

			// make sure we return a non-nil error, if we encounter any issues
			err = oerr

			wait := time.Duration(4+2*try) * time.Second
			if wait > 60*time.Second {
				wait = 60 * time.Second
			}

			time.Sleep(wait)

			continue
		}

		wsutil.WSDebug("Started after attempt:", try)

		return err
	}
}

// Open connects to the Websocket and authenticate it. You should usually use
// this function over Start().
func (g *Gateway) Open() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.OpenContext(ctx)
}

// OpenContext connects to the Websocket and authenticates it. You should
// usually use this function over Start(). The given context provides
// cancellation and timeout.
func (g *Gateway) OpenContext(ctx context.Context) error {
	// Reconnect to the Gateway
	if err := g.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	wsutil.WSDebug("Trying to start...")

	// Try to resume the connection
	if err := g.StartCtx(ctx); err != nil {
		return err
	}

	// Started successfully, return
	return nil
}

// Start calls StartCtx with a background context. You wouldn't usually use this
// function, but Open() instead.
func (g *Gateway) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.StartCtx(ctx)
}

// StartCtx authenticates with the websocket, or resume from a dead Websocket
// connection. You wouldn't usually use this function, but OpenCtx() instead.
func (g *Gateway) StartCtx(ctx context.Context) error {
	if err := g.start(ctx); err != nil {
		wsutil.WSDebug("Start failed:", err)

		// Close can be called with the mutex still acquired here, as the
		// pacemaker hasn't started yet.
		if err := g.Close(); err != nil {
			wsutil.WSDebug("Failed to close after start fail:", err)
		}
		return err
	}

	return nil
}

func (g *Gateway) start(ctx context.Context) error {
	// This is where we'll get our events
	ch := g.WS.Listen()

	// Create a new Hello event and wait for it.
	var hello HelloEvent
	// Wait for an OP 10 Hello.
	select {
	case e, ok := <-ch:
		if !ok {
			return errors.New("unexpected ws close while waiting for Hello")
		}
		if _, err := wsutil.AssertEvent(e, HelloOP, &hello); err != nil {
			return errors.Wrap(err, "error at Hello")
		}
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "failed to wait for Hello event")
	}

	wsutil.WSDebug("Hello received; duration:", hello.HeartbeatInterval)

	// Start the event handler, which also handles the pacemaker death signal.
	g.waitGroup.Add(1)

	// Use the pacemaker loop.
	g.PacerLoop.StartBeating(hello.HeartbeatInterval.Duration(), g, func(err error) {
		g.waitGroup.Done() // mark so Close() can exit.
		wsutil.WSDebug("Event loop stopped with error:", err)

		// Bail if there is no error or if the error is an explicit close, as
		// there might be an ongoing reconnection.
		if err == nil || errors.Is(err, wsutil.ErrWebsocketClosed) {
			return
		}

		// Only attempt to reconnect if we have a session ID at all. We may not
		// have one if we haven't even connected successfully once.
		if g.SessionID() != "" {
			g.ErrorLog(err)
			g.Reconnect()
		}
	})

	// Send Discord either the Identify packet (if it's a fresh connection), or
	// a Resume packet (if it's a dead connection).
	if g.SessionID() == "" {
		// SessionID is empty, so this is a completely new session.
		if err := g.IdentifyCtx(ctx); err != nil {
			return errors.Wrap(err, "failed to identify")
		}
	} else {
		if err := g.ResumeCtx(ctx); err != nil {
			return errors.Wrap(err, "failed to resume")
		}
	}

	// Expect either READY or RESUMED before continuing.
	wsutil.WSDebug("Waiting for either READY or RESUMED.")

	// WaitForEvent should
	err := wsutil.WaitForEvent(ctx, g, ch, func(op *wsutil.OP) bool {
		switch op.EventName {
		case "READY":
			wsutil.WSDebug("Found READY event.")
			return true
		case "RESUMED":
			wsutil.WSDebug("Found RESUMED event.")
			return true
		}
		return false
	})

	if err != nil {
		return errors.Wrap(err, "first error")
	}

	// Bind the event channel to the pacemaker loop.
	g.PacerLoop.SetEventChannel(ch)

	wsutil.WSDebug("Started successfully.")

	return nil
}

// SendCtx is a low-level function to send an OP payload to the Gateway. Most
// users shouldn't touch this, unless they know what they're doing.
func (g *Gateway) SendCtx(ctx context.Context, code OPCode, v interface{}) error {
	var op = wsutil.OP{
		Code: code,
	}

	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "failed to encode v")
		}

		op.Data = b
	}

	b, err := json.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "failed to encode payload")
	}

	// WS should already be thread-safe.
	return g.WS.SendCtx(ctx, b)
}
