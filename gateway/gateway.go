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

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/pkg/errors"
)

var (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = "6"
	Encoding = "json"
	// Compress = "zlib-stream"
)

var (
	ErrMissingForResume = errors.New("missing session ID or sequence for resuming")
	ErrWSMaxTries       = errors.New("max tries reached")
)

// GatewayBotData contains the GatewayURL as well as extra metadata on how to
// shard bots.
type GatewayBotData struct {
	URL        string             `json:"url"`
	Shards     int                `json:"shards,omitempty"`
	StartLimit *SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit is the information on the current session start limit. It's
// used in GatewayBotData.
type SessionStartLimit struct {
	Total      int                  `json:"total"`
	Remaining  int                  `json:"remaining"`
	ResetAfter discord.Milliseconds `json:"reset_after"`
}

// URL asks Discord for a Websocket URL to the Gateway.
func URL() (string, error) {
	var g GatewayBotData

	return g.URL, httputil.NewClient().RequestJSON(
		&g, "GET",
		EndpointGateway,
	)
}

// BotURL fetches the Gateway URL along with extra metadata. The token
// passed in will NOT be prefixed with Bot.
func BotURL(token string) (*GatewayBotData, error) {
	var g *GatewayBotData

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

	SessionID string

	Identifier *Identifier
	Sequence   *Sequence
	PacerLoop  *wsutil.PacemakerLoop

	ErrorLog func(err error) // default to log.Println

	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions.
	AfterClose func(err error) // noop by default

	// Mutex to hold off calls when the WS is not available. Doesn't block if
	// Start() is not called or Close() is called. Also doesn't block for
	// Identify or Resume.
	// available sync.RWMutex

	// Filled by methods, internal use
	waitGroup *sync.WaitGroup
}

// NewGateway starts a new Gateway with the default stdlib JSON driver. For more
// information, refer to NewGatewayWithDriver.
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

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() error {
	wsutil.WSDebug("Trying to close.")

	// Check if the WS is already closed:
	if g.waitGroup == nil && g.PacerLoop.Stopped() {
		wsutil.WSDebug("Gateway is already closed.")

		g.AfterClose(nil)
		return nil
	}

	// If the pacemaker is running:
	if !g.PacerLoop.Stopped() {
		wsutil.WSDebug("Stopping pacemaker...")

		// Stop the pacemaker and the event handler
		g.PacerLoop.Stop()

		wsutil.WSDebug("Stopped pacemaker.")
	}

	wsutil.WSDebug("Waiting for WaitGroup to be done.")

	// This should work, since Pacemaker should signal its loop to stop, which
	// would also exit our event loop. Both would be 2.
	g.waitGroup.Wait()

	// Mark g.waitGroup as empty:
	g.waitGroup = nil

	wsutil.WSDebug("WaitGroup is done. Closing the websocket.")

	err := g.WS.Close()
	g.AfterClose(err)
	return err
}

// Reconnect tries to reconnect forever. It will resume the connection if
// possible. If an Invalid Session is received, it will start a fresh one.
func (g *Gateway) Reconnect() error {
	return g.ReconnectContext(context.Background())
}

func (g *Gateway) ReconnectContext(ctx context.Context) error {
	wsutil.WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	g.Close()

	for i := 1; ; i++ {
		wsutil.WSDebug("Trying to dial, attempt", i)

		// Condition: err == ErrInvalidSession:
		// If the connection is rate limited (documented behavior):
		// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

		if err := g.OpenContext(ctx); err != nil {
			g.ErrorLog(errors.Wrap(err, "failed to open gateway"))
			continue
		}

		wsutil.WSDebug("Started after attempt:", i)
		return nil
	}
}

// Open connects to the Websocket and authenticate it. You should usually use
// this function over Start().
func (g *Gateway) Open() error {
	return g.OpenContext(context.Background())
}

func (g *Gateway) OpenContext(ctx context.Context) error {
	// Reconnect to the Gateway
	if err := g.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	wsutil.WSDebug("Trying to start...")

	// Try to resume the connection
	if err := g.Start(); err != nil {
		return err
	}

	// Started successfully, return
	return nil
}

// Start authenticates with the websocket, or resume from a dead Websocket
// connection. This function doesn't block. You wouldn't usually use this
// function, but Open() instead.
func (g *Gateway) Start() error {
	// g.available.Lock()
	// defer g.available.Unlock()

	if err := g.start(); err != nil {
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

func (g *Gateway) start() error {
	// This is where we'll get our events
	ch := g.WS.Listen()

	// Make a new WaitGroup for use in background loops:
	g.waitGroup = new(sync.WaitGroup)

	// Wait for an OP 10 Hello
	var hello HelloEvent
	if _, err := wsutil.AssertEvent(<-ch, HelloOP, &hello); err != nil {
		return errors.Wrap(err, "error at Hello")
	}

	// Send Discord either the Identify packet (if it's a fresh connection), or
	// a Resume packet (if it's a dead connection).
	if g.SessionID == "" {
		// SessionID is empty, so this is a completely new session.
		if err := g.Identify(); err != nil {
			return errors.Wrap(err, "failed to identify")
		}
	} else {
		if err := g.Resume(); err != nil {
			return errors.Wrap(err, "failed to resume")
		}
	}

	// Expect either READY or RESUMED before continuing.
	wsutil.WSDebug("Waiting for either READY or RESUMED.")

	// WaitForEvent should
	err := wsutil.WaitForEvent(g, ch, func(op *wsutil.OP) bool {
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

	// Use the pacemaker loop.
	g.PacerLoop = wsutil.NewLoop(hello.HeartbeatInterval.Duration(), ch, g)

	// Start the event handler, which also handles the pacemaker death signal.
	g.waitGroup.Add(1)

	g.PacerLoop.RunAsync(func(err error) {
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

func (g *Gateway) Send(code OPCode, v interface{}) error {
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
	return g.WS.Send(b)
}
