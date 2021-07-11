// Package wsutil provides abstractions around the Websocket, including rate
// limits.
package wsutil

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

var (
	// WSTimeout is the timeout for connecting and writing to the Websocket,
	// before Gateway cancels and fails.
	WSTimeout = 30 * time.Second
	// WSBuffer is the size of the Event channel. This has to be at least 1 to
	// make space for the first Event: Ready or Resumed.
	WSBuffer = 10
	// WSError is the default error handler
	WSError = func(err error) { log.Println("Gateway error:", err) }
	// WSDebug is used for extra debug logging. This is expected to behave
	// similarly to log.Println().
	WSDebug = func(v ...interface{}) {}
)

type Event struct {
	Data []byte

	// Error is non-nil if Data is nil.
	Error error
}

// Websocket is a wrapper around a websocket Conn with thread safety and rate
// limiting for sending and throttling.
type Websocket struct {
	mutex  sync.Mutex
	conn   Connection
	addr   string
	closed bool

	sendLimiter *rate.Limiter
	dialLimiter *rate.Limiter

	// Constants. These must not be changed after the Websocket instance is used
	// once, as they are not thread-safe.

	// Timeout for connecting and writing to the Websocket, uses default
	// WSTimeout (global).
	Timeout time.Duration
}

// New creates a default Websocket with the given address.
func New(addr string) *Websocket {
	return NewCustom(NewConn(), addr)
}

// NewCustom creates a new undialed Websocket.
func NewCustom(conn Connection, addr string) *Websocket {
	return &Websocket{
		conn:   conn,
		addr:   addr,
		closed: true,

		sendLimiter: NewSendLimiter(),
		dialLimiter: NewDialLimiter(),

		Timeout: WSTimeout,
	}
}

// Dial waits until the rate limiter allows then dials the websocket.
func (ws *Websocket) Dial(ctx context.Context) error {
	if ws.Timeout > 0 {
		tctx, cancel := context.WithTimeout(ctx, ws.Timeout)
		defer cancel()

		ctx = tctx
	}

	if err := ws.dialLimiter.Wait(ctx); err != nil {
		// Expired, fatal error
		return errors.Wrap(err, "failed to wait")
	}

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if !ws.closed {
		WSDebug("Old connection not yet closed while dialog; closing it.")
		ws.conn.Close()
	}

	if err := ws.conn.Dial(ctx, ws.addr); err != nil {
		return errors.Wrap(err, "failed to dial")
	}

	ws.closed = false

	// Reset the send limiter.
	ws.sendLimiter = NewSendLimiter()

	return nil
}

// Listen returns the inner event channel or nil if the Websocket connection is
// not alive.
func (ws *Websocket) Listen() <-chan Event {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.closed {
		return nil
	}

	return ws.conn.Listen()
}

// Send sends b over the Websocket without a timeout.
func (ws *Websocket) Send(b []byte) error {
	return ws.SendCtx(context.Background(), b)
}

// SendCtx sends b over the Websocket with a deadline. It closes the internal
// Websocket if the Send method errors out.
func (ws *Websocket) SendCtx(ctx context.Context, b []byte) error {
	WSDebug("Waiting for the send rate limiter...")

	if err := ws.sendLimiter.Wait(ctx); err != nil {
		WSDebug("Send rate limiter timed out.")
		return errors.Wrap(err, "SendLimiter failed")
	}

	WSDebug("Send has passed the rate limiting. Waiting on mutex.")

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	WSDebug("Mutex lock acquired.")

	if ws.closed {
		return ErrWebsocketClosed
	}

	if err := ws.conn.Send(ctx, b); err != nil {
		// We need to clean up ourselves if things are erroring out.
		WSDebug("Conn: Error while sending; closing the connection. Error:", err)
		ws.close(false)
		return err
	}

	return nil
}

// Close closes the websocket connection. It assumes that the Websocket is
// closed even when it returns an error. If the Websocket was already closed
// before, ErrWebsocketClosed will be returned.
func (ws *Websocket) Close() error {
	WSDebug("Conn: Acquiring mutex lock to close...")

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	WSDebug("Conn: Write mutex acquired")

	return ws.close(false)
}

func (ws *Websocket) CloseGracefully() error {
	WSDebug("Conn: Acquiring mutex lock to close...")

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	WSDebug("Conn: Write mutex acquired")

	return ws.close(true)
}

// close closes the Websocket without acquiring the mutex. Refer to Close for
// more information.
func (ws *Websocket) close(graceful bool) error {
	if ws.closed {
		WSDebug("Conn: Websocket is already closed.")
		return ErrWebsocketClosed
	}

	ws.closed = true

	if graceful {
		WSDebug("Conn: Closing gracefully")
		return ws.conn.CloseGracefully()
	}

	WSDebug("Conn: Closing")
	return ws.conn.Close()
}
