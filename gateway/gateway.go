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
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/internal/lazytime"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

var (
	Version  = api.Version
	Encoding = "json"
)

// deadbeatDuration is the duration that limits whether the gateway should
// resume or restart entirely. If it's less than this duration, then it's deemed
// resumable.
const deadbeatDuration = 15 * time.Minute

// CodeInvalidSequence is the code returned by Discord to signal that the given
// sequence number is invalid.
const CodeInvalidSequence = 4007

// CodeShardingRequired is the code returned by Discord to signal that the bot
// must reshard before proceeding. For more information, see
// https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-close-event-codes.
const CodeShardingRequired = 4011

// URL asks Discord for a Websocket URL to the Gateway.
func URL(ctx context.Context) (string, error) {
	return api.GatewayURL(ctx)
}

// BotURL fetches the Gateway URL along with extra metadata. The token
// passed in will NOT be prefixed with Bot.
func BotURL(ctx context.Context, token string) (*api.BotData, error) {
	return api.NewClient(token).WithContext(ctx).BotURL()
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

// State contains the gateway state. It is a piece of data that can be shared
// across gateways during construction to be used for resuming a connection or
// starting a new one with the previous data.
//
// The data structure itself is not thread-safe, so they may only be pulled from
// the gateway after it's done and set before it's done.
type State struct {
	Identifier Identifier
	SessionID  string
	Sequence   int64
}

// Gateway describes an instance that handles the Discord gateway. It is
// basically an abstracted concurrent event loop that the user could signal to
// start connecting to the Discord gateway server.
type Gateway struct {
	gateway *ws.Gateway
	state   State

	// non-mutex-guarded states
	// TODO: make lastBeat part of ws.Gateway so it can keep track of whether or
	// not the websocket is dead.
	beatMutex  sync.Mutex
	sentBeat   time.Time
	echoBeat   time.Time
	retryTimer lazytime.Timer
}

// NewWithIntents creates a new Gateway with the given intents and the default
// stdlib JSON driver. Refer to NewGatewayWithDriver and AddIntents.
func NewWithIntents(ctx context.Context, token string, intents ...Intents) (*Gateway, error) {
	var allIntents Intents
	for _, intent := range intents {
		allIntents |= intent
	}

	g, err := New(ctx, token)
	if err != nil {
		return nil, err
	}

	g.AddIntents(allIntents)
	return g, nil
}

// New creates a new Gateway to the default Discord server.
func New(ctx context.Context, token string) (*Gateway, error) {
	return NewWithIdentifier(ctx, DefaultIdentifier(token))
}

// NewWithIdentifier creates a new Gateway with the given gateway identifier and
// the default everything. Sharded bots should prefer this function for the
// shared identifier. The given Identifier will be modified.
func NewWithIdentifier(ctx context.Context, id Identifier) (*Gateway, error) {
	gatewayURL, err := id.QueryGateway(ctx)
	if err != nil {
		return nil, err
	}

	gatewayURL = AddGatewayParams(gatewayURL)
	gateway := NewCustomWithIdentifier(gatewayURL, id, nil)

	return gateway, nil
}

// NewCustom creates a new Gateway with a custom gateway URL and a new
// Identifier. Most bots connecting to the official server should not use these
// custom functions.
func NewCustom(gatewayURL, token string) *Gateway {
	return NewCustomWithIdentifier(gatewayURL, DefaultIdentifier(token), nil)
}

// DefaultGatewayOpts contains the default options to be used for connecting to
// the gateway.
var DefaultGatewayOpts = ws.GatewayOpts{
	ReconnectDelay: func(try int) time.Duration {
		// minimum 4 seconds
		return time.Duration(4+(2*try)) * time.Second
	},
	// FatalCloseCodes contains the default gateway close codes that will cause
	// the gateway to exit. In other words, it's a list of unrecoverable close
	// codes.
	FatalCloseCodes: []int{
		4004, // authentication failed
		4010, // invalid shard sent
		4011, // sharding required
		4012, // invalid API version
		4013, // invalid intents
		4014, // disallowed intents
	},
	DialTimeout:           0,
	ReconnectAttempt:      0,
	AlwaysCloseGracefully: true,
}

// NewCustomWithIdentifier creates a new Gateway with a custom gateway URL and a
// pre-existing Identifier. If opts is nil, then DefaultGatewayOpts is used.
func NewCustomWithIdentifier(gatewayURL string, id Identifier, opts *ws.GatewayOpts) *Gateway {
	return NewFromState(gatewayURL, State{Identifier: id}, opts)
}

// NewFromState creates a new gateway from the given state and optionally
// gateway options. If opts is nil, then DefaultGatewayOpts is used.
func NewFromState(gatewayURL string, state State, opts *ws.GatewayOpts) *Gateway {
	if opts == nil {
		opts = &DefaultGatewayOpts
	}

	gw := ws.NewGateway(ws.NewWebsocket(ws.NewCodec(OpUnmarshalers), gatewayURL), opts)
	return &Gateway{
		gateway: gw,
		state:   state,
	}
}

// Opts returns a copy of the gateway options that are being used.
func (g *Gateway) Opts() *ws.GatewayOpts {
	return g.gateway.Opts()
}

// State returns a copy of the gateway's internal state. It panics if the
// gateway is currently running.
func (g *Gateway) State() State {
	g.gateway.AssertIsNotRunning()
	return g.state
}

// SetState sets the gateway's state.
func (g *Gateway) SetState(state State) {
	g.gateway.AssertIsNotRunning()
	g.state = state
}

// AddIntents adds a Gateway Intent before connecting to the Gateway. This
// function will only work before Connect() is called. Calling it once Connect()
// is called will result in a panic.
func (g *Gateway) AddIntents(i Intents) {
	g.gateway.AssertIsNotRunning()
	g.state.Identifier.AddIntents(i)
}

// SentBeat returns the last time that the heart was beaten. If the gateway has
// never connected, then a zero-value time is returned.
func (g *Gateway) SentBeat() time.Time {
	g.beatMutex.Lock()
	defer g.beatMutex.Unlock()

	return g.sentBeat
}

// EchoBeat returns the last time that the heartbeat was acknowledged. It is
// similar to SentBeat.
func (g *Gateway) EchoBeat() time.Time {
	g.beatMutex.Lock()
	defer g.beatMutex.Unlock()

	return g.echoBeat
}

// Latency is a convenient function around SentBeat and EchoBeat. It subtracts
// the EchoBeat with the SentBeat.
func (g *Gateway) Latency() time.Duration {
	g.beatMutex.Lock()
	defer g.beatMutex.Unlock()

	return g.echoBeat.Sub(g.sentBeat)
}

// LastError returns the last error that the gateway has received. It only
// returns a valid error if the gateway's event loop as exited. If the event
// loop hasn't been started AND stopped, the function will panic.
func (g *Gateway) LastError() error {
	return g.gateway.LastError()
}

// Send is a function to send an Op payload to the Gateway.
func (g *Gateway) Send(ctx context.Context, data ws.Event) error {
	return g.gateway.Send(ctx, data)
}

// Connect starts the background goroutine that tries its best to maintain a
// stable connection to the Discord gateway. To the user, the gateway should
// appear to be working seamlessly.
//
// # Behaviors
//
// There are several behaviors that the gateway will overload onto the channel.
//
// Once the gateway has exited, fatally or not, the event channel returned by
// Connect will be closed. The user should therefore know whether or not the
// gateway has exited by spinning on the channel until it is closed.
//
// If Connect is called twice, the second call will return the same exact
// channel that the first call has made without starting any new goroutines,
// except if the gateway is already closed, then a new gateway will spin up with
// the existing gateway state.
//
// If the gateway stumbles upon any background errors, it will do its best to
// recover from it, but errors will be notified to the user using the
// BackgroundErrorEvent event. The user can type-assert the Op's data field,
// like so:
//
//	switch data := ev.Data.(type) {
//	case *gateway.BackgroundErrorEvent:
//	    log.Println("gateway error:", data.Error)
//	}
//
// # Closing
//
// As outlined in the first paragraph, closing the gateway would involve
// cancelling the context that's given to gateway. If AlwaysCloseGracefully is
// true (which it is by default), then the gateway is closed gracefully, and the
// session ID is invalidated.
//
// To wait until the gateway has completely successfully exited, the user can
// keep spinning on the event loop:
//
//	for op := range ch {
//	    select op.Data.(type) {
//	    case *gateway.ReadyEvent:
//	        // Close the gateway on READY.
//	        cancel()
//	    }
//	}
//
//	// Gateway is now completely closed.
//
// To capture the final close errors, the user can use the Error method once the
// event channel is closed, like so:
//
//	var err error
//
//	for op := range ch {
//	    switch data := op.Data.(type) {
//	    case *gateway.ReadyEvent:
//	        cancel()
//	    }
//	}
//
//	// Gateway is now completely closed.
//	if gateway.LastError() != nil {
//	    return gateway.LastError()
//	}
func (g *Gateway) Connect(ctx context.Context) <-chan ws.Op {
	return g.gateway.Connect(ctx, &gatewayImpl{Gateway: g})
}

type gatewayImpl struct {
	*Gateway
	heartrate    time.Duration
	lastSentBeat time.Time
}

func (g *gatewayImpl) invalidate() {
	g.state.SessionID = ""
	g.state.Sequence = 0
}

// sendIdentify sends off the Identify command with the Gateway's IdentifyData
// with the given context for timeout.
func (g *gatewayImpl) sendIdentify(ctx context.Context) error {
	if err := g.state.Identifier.Wait(ctx); err != nil {
		return fmt.Errorf("can't wait for identify(): %w", err)
	}

	return g.gateway.Send(ctx, &g.state.Identifier.IdentifyCommand)
}

func (g *gatewayImpl) sendResume(ctx context.Context) error {
	return g.gateway.Send(ctx, &ResumeCommand{
		Token:     g.state.Identifier.Token,
		SessionID: g.state.SessionID,
		Sequence:  g.state.Sequence,
	})
}

func (g *gatewayImpl) OnOp(ctx context.Context, op ws.Op) bool {
	if op.Code == dispatchOp {
		g.state.Sequence = op.Sequence
	}

	switch data := op.Data.(type) {
	case *ws.CloseEvent:
		if data.Code == CodeInvalidSequence {
			// Invalid sequence.
			g.invalidate()
		}

		g.gateway.QueueReconnect()

	case *HelloEvent:
		g.heartrate = data.HeartbeatInterval.Duration()
		g.gateway.ResetHeartbeat(g.heartrate)

		now := time.Now()

		g.beatMutex.Lock()
		// Determine that we shouldn't reconnect if the last time we've received
		// a heart beat was over (deadbeatDuration) ago.
		resumable := g.echoBeat.IsZero() || time.Since(g.echoBeat) < deadbeatDuration
		// Reset gateway times.
		g.echoBeat = time.Time{}
		g.sentBeat = time.Time{}
		// Set the last sent beat time so we can treat sending an Identify or
		// Resume as sending a heartbeat.
		g.lastSentBeat = now
		g.beatMutex.Unlock()

		// Send Discord either the Identify packet (if it's a fresh
		// connection), or a Resume packet (if it's a dead connection).
		if !resumable || g.state.SessionID == "" || g.state.Sequence == 0 {
			// SessionID is empty, so this is a completely new session.
			if err := g.sendIdentify(ctx); err != nil {
				g.gateway.SendErrorWrap(err, "failed to send identify")
				g.gateway.QueueReconnect()
			}
		} else {
			if err := g.sendResume(ctx); err != nil {
				g.gateway.SendErrorWrap(err, "failed to send resume")
				g.gateway.QueueReconnect()
			}
		}

	case *InvalidSessionEvent:
		// Wipe the session state.
		g.invalidate()

		if !*data {
			g.gateway.QueueReconnect()
			break
		}

		// Discord expects us to wait before reconnecting.
		g.retryTimer.Reset(time.Duration(rand.Intn(5)+1) * time.Second)
		if err := g.retryTimer.Wait(ctx); err != nil {
			g.gateway.SendErrorWrap(err, "failed to wait before identifying")
			g.gateway.QueueReconnect()
			break
		}

		// If we fail to identify, then the gateway cannot continue with
		// a bad identification, since it's likely a user error.
		if err := g.sendIdentify(ctx); err != nil {
			g.gateway.SendErrorWrap(err, "failed to identify")
			g.gateway.QueueReconnect()
			break
		}

	case *HeartbeatCommand:
		g.SendHeartbeat(ctx)

	case *HeartbeatAckEvent:
		g.useLastSentBeat()

	case *ReconnectEvent:
		g.gateway.QueueReconnect()

	case *ReadyEvent:
		g.state.SessionID = data.SessionID
		g.useLastSentBeat()

	case *ResumedEvent:
		g.useLastSentBeat()
	}

	return true
}

func (g *gatewayImpl) useLastSentBeat() {
	now := time.Now()

	g.beatMutex.Lock()
	g.sentBeat = g.lastSentBeat
	g.echoBeat = now
	g.beatMutex.Unlock()
}

func (g *gatewayImpl) isDead() bool {
	if g.heartrate == 0 {
		return false
	}

	g.beatMutex.Lock()
	defer g.beatMutex.Unlock()

	if g.echoBeat.IsZero() {
		// No ack received yet. We wait for a bit.
		return false
	}

	// Allow 2 beats to miss before we declare dead.
	return g.lastSentBeat.Sub(g.echoBeat) > 2*g.heartrate
}

// SendHeartbeat sends a heartbeat with the gateway's current sequence.
func (g *gatewayImpl) SendHeartbeat(ctx context.Context) {
	g.lastSentBeat = time.Now()

	// TODO: move this to ws.Gateway
	if g.isDead() {
		g.gateway.SendError(fmt.Errorf("heartbeat timed out"))
		g.gateway.QueueReconnect()
		return
	}

	sequence := HeartbeatCommand(g.state.Sequence)
	if err := g.gateway.Send(ctx, &sequence); err != nil {
		g.gateway.SendErrorWrap(err, "heartbeat error")
		g.gateway.QueueReconnect()
		return
	}
}

// Close closes the state.
func (g *gatewayImpl) Close() error {
	g.retryTimer.Stop()
	g.invalidate()
	return nil
}
