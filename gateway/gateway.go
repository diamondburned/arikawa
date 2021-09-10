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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	Version  = api.Version
	Encoding = "json"
)

var (
	ErrMissingForResume = errors.New("missing session ID or sequence for resuming")
	ErrWSMaxTries       = errors.New(
		"could not connect to the Discord gateway before reaching the timeout")
	ErrClosed = errors.New("the gateway is closed and cannot reconnect")
)

// see
// https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-close-event-codes
const errCodeShardingRequired = 4011

// URL asks Discord for a Websocket URL to the Gateway.
func URL() (string, error) {
	return api.GatewayURL()
}

// BotURL fetches the Gateway URL along with extra metadata. The token
// passed in will NOT be prefixed with Bot.
func BotURL(token string) (*api.BotData, error) {
	return api.NewClient(token).BotURL()
}

// AddGatewayParams appends into the given URL string the gateway URL
// parameters.
func AddGatewayParams(baseURL string) string {
	param := url.Values{
		"v":        {Version},
		"encoding": {Encoding},
	}

	return baseURL + "?" + param.Encode()
}

type Gateway struct {
	WS *wsutil.Websocket

	// WSTimeout is a timeout for an arbitrary action. An example of this is the
	// timeout for Start and the timeout for sending each Gateway command
	// independently.
	WSTimeout time.Duration

	// ReconnectAttempts are the amount of attempts made to Reconnect, before
	// aborting. If this set to 0, unlimited attempts will be made.
	ReconnectAttempts uint

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

	// AfterClose is called after each close or pause. It is used mainly for
	// reconnections or any type of connection interruptions.
	//
	// Constructors will use a no-op function by default.
	AfterClose func(err error)

	onShardingRequired func()

	waitGroup sync.WaitGroup
	closed    chan struct{}
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

// NewGateway creates a new Gateway to the default Discord server.
func NewGateway(token string) (*Gateway, error) {
	return NewIdentifiedGateway(DefaultIdentifier(token))
}

// NewIdentifiedGateway creates a new Gateway with the given gateway identifier
// and the default everything. Sharded bots should prefer this function for the
// shared identifier.
func NewIdentifiedGateway(id *Identifier) (*Gateway, error) {
	var gatewayURL string
	var botData *api.BotData
	var err error

	if strings.HasPrefix(id.Token, "Bot ") {
		botData, err = BotURL(id.Token)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bot data")
		}
		gatewayURL = botData.URL

	} else {
		gatewayURL, err = URL()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get gateway endpoint")
		}
	}

	gatewayURL = AddGatewayParams(gatewayURL)
	gateway := NewCustomIdentifiedGateway(gatewayURL, id)

	// Use the supplied connect rate limit, if any.
	if botData != nil && botData.StartLimit != nil {
		resetAt := time.Now().Add(botData.StartLimit.ResetAfter.Duration())
		limiter := gateway.Identifier.IdentifyGlobalLimit

		// Update the burst to be the current given time and reset it back to
		// the default when the given time is reached.
		limiter.SetBurst(botData.StartLimit.Remaining)
		limiter.SetBurstAt(resetAt, botData.StartLimit.Total)

		// Update the maximum number of identify requests allowed per 5s.
		gateway.Identifier.IdentifyShortLimit.SetBurst(botData.StartLimit.MaxConcurrency)
	}

	return gateway, nil
}

// NewCustomGateway creates a new Gateway with a custom gateway URL and a new
// Identifier. Most bots connecting to the official server should not use these
// custom functions.
func NewCustomGateway(gatewayURL, token string) *Gateway {
	return NewCustomIdentifiedGateway(gatewayURL, DefaultIdentifier(token))
}

// NewCustomIdentifiedGateway creates a new Gateway with a custom gateway URL
// and a pre-existing Identifier. Refer to NewCustomGateway.
func NewCustomIdentifiedGateway(gatewayURL string, id *Identifier) *Gateway {
	return &Gateway{
		WS:        wsutil.NewCustom(wsutil.NewConn(), gatewayURL),
		WSTimeout: wsutil.WSTimeout,

		Events:     make(chan Event, wsutil.WSBuffer),
		Identifier: id,
		Sequence:   moreatomic.NewInt64(0),

		ErrorLog:   wsutil.WSError,
		AfterClose: func(error) {},
	}
}

// AddIntents adds a Gateway Intent before connecting to the Gateway. As such,
// this function will only work before Open() is called.
func (g *Gateway) AddIntents(i Intents) {
	if g.Identifier.Intents == nil {
		g.Identifier.Intents = option.NewUint(uint(i))
	} else {
		*g.Identifier.Intents |= uint(i)
	}
}

// HasIntents reports if the Gateway has the passed Intents.
//
// If no intents are set, i.e. if using a user account HasIntents will always
// return true.
func (g *Gateway) HasIntents(intents Intents) bool {
	if g.Identifier.Intents == nil {
		return true
	}

	return Intents(*g.Identifier.Intents).Has(intents)
}

// Close closes the underlying Websocket connection, invalidating the session
// ID.
//
// It will send a closing frame before ending the connection, closing it
// gracefully. This will cause the bot to appear as offline instantly.
func (g *Gateway) Close() error {
	return g.close(true)
}

// Pause pauses the Gateway connection, by ending the connection without
// sending a closing frame. This allows the connection to be resumed at a later
// point, by calling Reconnect or ReconnectCtx.
func (g *Gateway) Pause() error {
	return g.close(false)
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

	if graceful {
		// If a Reconnect is in progress, signal to cancel.
		close(g.closed)

		// Delete our session id, as we just invalidated it.
		g.sessionMu.Lock()
		g.sessionID = ""
		g.sessionMu.Unlock()
	}

	return err
}

// SessionID returns the session ID received after Ready. This function is
// concurrently safe.
func (g *Gateway) SessionID() string {
	g.sessionMu.RLock()
	defer g.sessionMu.RUnlock()

	return g.sessionID
}

// UseSessionID overrides the internal session ID for the one the user provides.
func (g *Gateway) UseSessionID(sessionID string) {
	g.sessionMu.Lock()
	defer g.sessionMu.Unlock()

	g.sessionID = sessionID
}

// OnShardingRequired sets the function to be called if Discord closes with
// error code 4011 aka Sharding Required. When called, the Gateway will already
// be closed, and can (after increasing the number of shards) be reopened using
// Open. Reconnect or ReconnectCtx, however, will not be available as the
// session is invalidated.
//
// The gateway will completely halt what it's doing in the background when this
// callback is called.
func (g *Gateway) OnShardingRequired(fn func()) {
	g.onShardingRequired = fn
}

// Reconnect tries to reconnect to the Gateway until the ReconnectAttempts are
// reached.
func (g *Gateway) Reconnect() {
	g.ReconnectCtx(context.Background())
}

// ReconnectCtx attempts to Reconnect until context expires.
// If the context expires FatalErrorCallback will be called with ErrWSMaxTries,
// and the last error returned by Open will be returned.
func (g *Gateway) ReconnectCtx(ctx context.Context) (err error) {
	wsutil.WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	g.Pause()

	for try := uint(1); g.ReconnectAttempts == 0 || g.ReconnectAttempts >= try; try++ {
		select {
		case <-g.closed:
			g.ErrorLog(ErrClosed)
			return ErrClosed
		case <-ctx.Done():
			wsutil.WSDebug("Unable to Reconnect after", try, "attempts, aborting")
			g.FatalErrorCallback(ErrWSMaxTries)
			return err
		default:
		}

		wsutil.WSDebug("Trying to dial, attempt", try)

		// if we encounter an error, make sure we return it, and not nil
		if oerr := g.Open(ctx); oerr != nil {
			err = oerr
			g.ErrorLog(oerr)

			wait := time.Duration(4+2*try) * time.Second
			if wait > 60*time.Second {
				wait = 60 * time.Second
			}

			time.Sleep(wait)
			continue
		}

		wsutil.WSDebug("Started after attempt:", try)
		return nil
	}

	wsutil.WSDebug("Unable to Reconnect after", g.ReconnectAttempts, "attempts, aborting")
	return err
}

// Open connects to the Websocket and authenticates it. You should usually use
// this function over Start(). The given context provides cancellation and
// timeout.
func (g *Gateway) Open(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	// Reconnect to the Gateway
	if err := g.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "failed to Reconnect")
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
	g.closed = make(chan struct{})

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

		if err != nil && g.onShardingRequired != nil {
			// If Discord signals us sharding is required, do not attempt to
			// Reconnect, unless we don't know what to do. Instead invalidate
			// our session ID, as we cannot resume, call OnShardingRequired, and
			// exit.
			var cerr *websocket.CloseError
			if errors.As(err, &cerr) && cerr.Code == errCodeShardingRequired {
				g.ErrorLog(cerr)
				g.UseSessionID("")
				g.onShardingRequired()
				return
			}
		}

		// Bail if there is no error or if the error is an explicit close, as
		// there might be an ongoing reconnection.
		if err == nil || errors.Is(err, wsutil.ErrWebsocketClosed) {
			return
		}

		// Only attempt to Reconnect if we have a session ID at all. We may not
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

	// WaitForEvent should until the bot becomes ready or resumes (if a
	// previous ready event has already been called).
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
