package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"errors"
	"github.com/diamondburned/arikawa/v3/internal/lazytime"
	"github.com/diamondburned/arikawa/v3/utils/json"
)

// ConnectionError is given to the user if the gateway fails to connect to the
// gateway for any reason, including during an initial connection or a
// reconnection. To check for this error, use the errors.As function.
type ConnectionError struct {
	Err error
}

// Unwrap unwraps the ConnectionError.
func (err ConnectionError) Unwrap() error { return err.Err }

// Error formats the error.
func (err ConnectionError) Error() string {
	return fmt.Sprintf("error reconnecting: %s", err.Err)
}

// BackgroundErrorEvent describes an error that the gateway event loop might
// stumble upon while it's running. See Gateway's documentation for possible
// usages.
type BackgroundErrorEvent struct {
	Err error
}

var _ Event = (*BackgroundErrorEvent)(nil)

// Unwrap returns err.Err.
func (err *BackgroundErrorEvent) Unwrap() error { return err.Err }

// Error formats the BackgroundErrorEvent.
func (err *BackgroundErrorEvent) Error() string {
	return "background gateway error: " + err.Err.Error()
}

// Op implements Op. It returns -1.
func (err *BackgroundErrorEvent) Op() OpCode { return -1 }

// EventType implements Op. It returns an opaque unique string.
func (err *BackgroundErrorEvent) EventType() EventType {
	return "__ws.BackgroundErrorEvent"
}

// GatewayOpts describes the gateway event loop options.
type GatewayOpts struct {
	// ReconnectDelay determines the duration to idle after each failed retry.
	// This can be used to implement exponential backoff. The default is already
	// sane, so this field rarely needs to be changed.
	ReconnectDelay func(try int) time.Duration

	// FatalCloseCodes is a list of close codes that will cause the gateway to
	// exit out if it stumbles on one of these. It is a copy of FatalCloseCodes
	// (the global variable) by default.
	FatalCloseCodes []int

	// DialTimeout is the timeout to wait for each websocket dial before failing
	// it and retrying. Default is 0.
	DialTimeout time.Duration

	// ReconnectAttempt is the maximum number of attempts made to Reconnect
	// before aborting the whole gateway. If this set to 0, unlimited attempts
	// will be made. Default is 0.
	ReconnectAttempt int

	// AlwaysCloseGracefully, if true, will always make the Gateway close
	// gracefully once the context given to Open is cancelled. It governs the
	// Close behavior. The default is true.
	AlwaysCloseGracefully bool
}

// DefaultGatewayOpts is the default event loop options.
var DefaultGatewayOpts = GatewayOpts{
	ReconnectDelay: func(try int) time.Duration {
		// minimum 4 seconds
		return time.Duration(4+(2*try)) * time.Second
	},
	DialTimeout:           0,
	ReconnectAttempt:      0,
	AlwaysCloseGracefully: true,
}

// ErrorIsFatalClose returns true if the error is a fatal close error. It uses
// opts.FatalCloseCodes to check for the codes.
func (opts GatewayOpts) ErrorIsFatalClose(err error) bool {
	var closeErr *CloseEvent
	if !errors.As(err, &closeErr) {
		return false
	}

	for _, code := range opts.FatalCloseCodes {
		if code == closeErr.Code {
			return true
		}
	}

	return false
}

// Gateway describes an instance that handles the Discord gateway. It is
// basically an abstracted concurrent event loop that the user could signal to
// start connecting to the Discord gateway server.
type Gateway struct {
	ws *Websocket

	reconnect chan struct{}
	heart     lazytime.Ticker
	srcOp     <-chan Op // from WS
	outer     outerState
	lastError error

	opts GatewayOpts
}

// outerState holds gateway state that the caller may change concurrently. As
// such, it holds a mutex to allow that. The main purpose of this
// synchronization is to allow the caller to use the gateway while the event
// loop is still running without having the event loop muddle in without locking
// properly. For example, opCh is given to the event loop as a copy; the event
// loop must never access the outerState directly.
type outerState struct {
	sync.Mutex
	ch      chan Op
	started bool
}

// Handler describes a gateway handler. It describes the core that governs the
// behavior of the gateway event loop.
type Handler interface {
	// OnOp is called by the gateway event loop on every new Op. If the returned
	// boolean is false, then the loop fatally exits.
	OnOp(context.Context, Op) (canContinue bool)
	// SendHeartbeat is called by the gateway event loop everytime a heartbeat
	// needs to be sent over.
	SendHeartbeat(context.Context)
	// Close closes the handler.
	Close() error
}

// NewGateway creates a new Gateway with a custom gateway URL and a pre-existing
// Identifier. If opts is nil, then DefaultOpts is used.
func NewGateway(ws *Websocket, opts *GatewayOpts) *Gateway {
	if opts == nil {
		opts = &DefaultGatewayOpts
	}

	return &Gateway{
		ws:   ws,
		opts: *opts,
	}
}

// Opts returns a copy of the gateway options. The options can only be changed
// during construction, so a copy is a must.
func (g *Gateway) Opts() *GatewayOpts {
	cpy := g.opts
	return &cpy
}

// Send is a function to send an Op payload to the Gateway.
func (g *Gateway) Send(ctx context.Context, data Event) error {
	op := Op{
		Code: data.Op(),
		Type: data.EventType(),
		Data: data,
	}

	WSDebug("sending command Op", op.Code, "type", op.Type)

	b, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	// WS should already be thread-safe.
	return g.ws.Send(ctx, b)
}

// HasStarted returns true if the gateway event loop is currently spinning.
func (g *Gateway) HasStarted() bool {
	g.outer.Lock()
	defer g.outer.Unlock()

	return g.outer.started
}

// AssertIsNotRunning asserts that the gateway is currently not running. If the
// gateway is running, the method will panic. Since a gateway cannot be started
// back up, this method can be used to detect whether or not the caller in a
// single goroutine can read the state safely.
func (g *Gateway) AssertIsNotRunning() {
	g.outer.Lock()
	defer g.outer.Unlock()

	if !g.outer.started {
		return
	}

	// Hack to ensure that the event channel is closed.
	select {
	case _, ok := <-g.outer.ch:
		if !ok {
			return
		}
		// The panic behavior is a must, because if this branch is hit, then
		// we've actually stolen an event from the channel unexpectedly, putting
		// the event loop under a weird state.
		//
		// An alternative solution to this bug would be to mutex-guard the error
		// field, but the purpose of this method isn't to be called before the
		// gateway has been stopped.
		panic("ws: Error called while Gateway is still running")
	default:
		panic("ws: Error called while Gateway is still running")
	}
}

// Connect starts the background goroutine that tries its best to maintain a
// stable connection to the Websocket gateway. To the user, the gateway should
// appear to be working seamlessly.
//
// For more documentation, refer to (*gateway.Gateway).Connect.
func (g *Gateway) Connect(ctx context.Context, h Handler) <-chan Op {
	g.outer.Lock()
	defer g.outer.Unlock()

	if !g.outer.started {
		g.outer.started = true
		g.outer.ch = make(chan Op, 1)
		go g.spin(ctx, h)
	}

	return g.outer.ch
}

// LastError returns the last error that the gateway has received.
func (g *Gateway) LastError() error {
	g.AssertIsNotRunning()
	return g.lastError
}

// finalize closes the gateway permanently.
func (g *Gateway) finalize(h Handler) {
	var err error

	if g.opts.AlwaysCloseGracefully {
		err = g.ws.CloseGracefully()
	} else {
		err = g.ws.Close()
	}

	if err != nil {
		g.SendErrorWrap(err, "failed to finalize websocket")
	}

	if err := h.Close(); err != nil {
		g.SendError(err)
	}

	g.outer.Lock()
	close(g.outer.ch)
	g.outer.started = false
	g.outer.Unlock()
}

// QueueReconnect queues a reconnection in the gateway loop. This method should
// only be called in the event loop ONCE; calling more than once will deadlock
// the loop.
func (g *Gateway) QueueReconnect() {
	select {
	case g.reconnect <- struct{}{}:
	default:
	}

	g.heart.Stop()
}

// ResetHeartbeat resets the heartbeat to be the given duration.
func (g *Gateway) ResetHeartbeat(d time.Duration) {
	g.heart.Reset(d)
}

// SendError sends the given error wrapped in a BackgroundErrorEvent into the
// event channel.
func (g *Gateway) SendError(err error) {
	event := &BackgroundErrorEvent{err}

	g.outer.ch <- Op{
		Code: event.Op(),
		Type: event.EventType(),
		Data: event,
	}
	g.lastError = err
}

// SendErrorWrap is a convenient function over SendError.
func (g *Gateway) SendErrorWrap(err error, message string) {
	g.SendError(fmt.Errorf("%s: %w", message, err))
}

func (g *Gateway) spin(ctx context.Context, h Handler) {
	// Always close the event channel once we exit.
	defer g.finalize(h)

	var retryTimer lazytime.Timer
	defer retryTimer.Stop()

	g.reconnect = make(chan struct{}, 1)
	g.reconnect <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return

		case op, ok := <-g.srcOp:
			if !ok {
				// Skip zero-value Ops that may happen on gateway closure.
				continue
			}

			switch data := op.Data.(type) {
			case *CloseEvent:
				if g.opts.ErrorIsFatalClose(data) {
					// Don't wrap the error, but instead, just pipe it as-is
					// through the channel.
					g.outer.ch <- op
					g.lastError = data
					return
				}
			}

			ok = h.OnOp(ctx, op)
			g.outer.ch <- op
			if !ok {
				return
			}

			// Everything went well. Invalidate the error.
			g.lastError = nil

		case <-g.heart.C:
			h.SendHeartbeat(ctx)

		case <-g.reconnect:
			// Close the previous connection if it's not already. Ignore the
			// already closed error.
			if err := g.ws.Close(); err != nil && !errors.Is(err, ErrWebsocketClosed) {
				g.SendErrorWrap(err, "error closing before reconnecting")
			}

			// Invalidate our srcOp.
			g.srcOp = nil

			// Keep track of the last error for notifying.
			var err error

		retryLoop:
			for try := 0; g.opts.ReconnectAttempt == 0 || try < g.opts.ReconnectAttempt; try++ {
				g.srcOp, err = g.ws.Dial(ctx)
				if err == nil {
					break
				}

				// Exit if the context expired.
				select {
				case <-ctx.Done():
					err = ctx.Err()
					break retryLoop
				default:
				}

				// Signal an error before retrying.
				g.SendError(ConnectionError{err})

				retryTimer.Reset(g.opts.ReconnectDelay(try))
				if err := retryTimer.Wait(ctx); err != nil {
					g.SendError(ConnectionError{ctx.Err()})
					return
				}
			}

			// Ensure that we've reconnected successfully. Exit otherwise.
			if g.srcOp == nil {
				err = fmt.Errorf("failed to reconnect after max attempts: %w", err)
				g.SendError(ConnectionError{err})
				return
			}
		}
	}
}
