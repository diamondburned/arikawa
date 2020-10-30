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
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/pkg/errors"
)

var (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = "8"
	Encoding = "json"
	// Compress = "zlib-stream"
)

var (
	ErrMissingForResume = errors.New("missing session ID or sequence for resuming")
	ErrWSMaxTries       = errors.New("max tries reached")
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
	WS        *wsutil.Websocket
	WSTimeout time.Duration

	// All events sent over are pointers to Event structs (structs suffixed with
	// "Event"). This shouldn't be accessed if the Gateway is created with a
	// Session.
	Events chan Event

	// SessionID is used to store the session ID received after Ready. It is not
	// thread-safe.
	SessionID string

	Identifier *Identifier
	Sequence   *Sequence

	PacerLoop wsutil.PacemakerLoop

	ErrorLog func(err error) // default to log.Println

	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions.
	AfterClose func(err error) // noop by default

	// Filled by methods, internal use
	waitGroup *sync.WaitGroup
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
		Sequence:   NewSequence(),

		ErrorLog:   wsutil.WSError,
		AfterClose: func(error) {},
	}
}

// AddIntents adds a Gateway Intent before connecting to the Gateway. As such,
// this function will only work before Open() is called.
func (g *Gateway) AddIntents(i Intents) {
	g.Identifier.Intents |= i
}

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() (err error) {
	wsutil.WSDebug("Trying to close.")

	// Check if the WS is already closed:
	if g.PacerLoop.Stopped() {
		wsutil.WSDebug("Gateway is already closed.")
		return err
	}

	// Trigger the close callback on exit.
	defer func() { g.AfterClose(err) }()

	// If the pacemaker is running:
	if !g.PacerLoop.Stopped() {
		wsutil.WSDebug("Stopping pacemaker...")

		// Stop the pacemaker and the event handler.
		g.PacerLoop.Stop()

		wsutil.WSDebug("Stopped pacemaker.")
	}

	wsutil.WSDebug("Closing the websocket...")
	err = g.WS.Close()

	wsutil.WSDebug("Waiting for WaitGroup to be done.")

	// This should work, since Pacemaker should signal its loop to stop, which
	// would also exit our event loop. Both would be 2.
	g.waitGroup.Wait()

	wsutil.WSDebug("WaitGroup is done. Closing the websocket.")
	return err
}

// Reconnect tries to reconnect forever. It will resume the connection if
// possible. If an Invalid Session is received, it will start a fresh one.
func (g *Gateway) Reconnect() {
	for {
		if err := g.ReconnectCtx(context.Background()); err != nil {
			g.ErrorLog(err)
		} else {
			return
		}
	}
}

// ReconnectCtx attempts to reconnect until context expires. If context cannot
// expire, then the gateway will try to reconnect forever.
func (g *Gateway) ReconnectCtx(ctx context.Context) (err error) {
	wsutil.WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	g.Close()

	for i := 1; ; i++ {
		select {
		case <-ctx.Done():
			return err
		default:
		}

		wsutil.WSDebug("Trying to dial, attempt", i)

		// Condition: err == ErrInvalidSession:
		// If the connection is rate limited (documented behavior):
		// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

		// make sure we don't overwrite our last error
		if err = g.OpenContext(ctx); err != nil {
			g.ErrorLog(err)
			continue
		}

		wsutil.WSDebug("Started after attempt:", i)

		return
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

	// Make a new WaitGroup for use in background loops:
	g.waitGroup = new(sync.WaitGroup)

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

	// Send Discord either the Identify packet (if it's a fresh connection), or
	// a Resume packet (if it's a dead connection).
	if g.SessionID == "" {
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

	// Start the event handler, which also handles the pacemaker death signal.
	g.waitGroup.Add(1)

	// Use the pacemaker loop.
	g.PacerLoop.RunAsync(hello.HeartbeatInterval.Duration(), ch, g, func(err error) {
		g.waitGroup.Done() // mark so Close() can exit.
		wsutil.WSDebug("Event loop stopped with error:", err)

		if err != nil {
			g.ErrorLog(err)
			g.Reconnect()
		}
	})

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
