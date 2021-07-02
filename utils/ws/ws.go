// Package wsutil provides abstractions around the Websocket, including rate
// limits.
package ws

import (
	"context"
	"log"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

var (
	// WSError is the default error handler
	WSError = func(err error) { log.Println("Gateway error:", err) }
	// WSDebug is used for extra debug logging. This is expected to behave
	// similarly to log.Println().
	WSDebug = func(v ...interface{}) {}
)

// Websocket is a wrapper around a websocket Conn with thread safety and rate
// limiting for sending and throttling.
type Websocket struct {
	mutex sync.Mutex
	conn  Connection
	addr  string

	// If you ever need access to these fields from outside the package, please
	// open an issue. It might be worth it to refactor these out for distributed
	// sharding.

	sendLimiter *rate.Limiter
	dialLimiter *rate.Limiter
}

// NewWebsocket creates a default Websocket with the given address.
func NewWebsocket(c Codec, addr string) *Websocket {
	return NewCustomWebsocket(NewConn(c), addr)
}

// NewCustomWebsocket creates a new undialed Websocket.
func NewCustomWebsocket(conn Connection, addr string) *Websocket {
	return &Websocket{
		conn: conn,
		addr: addr,

		sendLimiter: NewSendLimiter(),
		dialLimiter: NewDialLimiter(),
	}
}

// Dial waits until the rate limiter allows then dials the websocket.
func (ws *Websocket) Dial(ctx context.Context) (<-chan Op, error) {
	if err := ws.dialLimiter.Wait(ctx); err != nil {
		// Expired, fatal error
		return nil, errors.Wrap(err, "failed to wait for dial rate limiter")
	}

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	// Reset the send limiter.
	// TODO: see if each limit only applies to one connection or not.
	ws.sendLimiter = NewSendLimiter()

	return ws.conn.Dial(ctx, ws.addr)
}

// Send sends b over the Websocket with a deadline. It closes the internal
// Websocket if the Send method errors out.
func (ws *Websocket) Send(ctx context.Context, b []byte) error {
	WSDebug("Acquiring the websocket mutex for sending.")

	ws.mutex.Lock()
	sendLimiter := ws.sendLimiter
	conn := ws.conn
	ws.mutex.Unlock()

	WSDebug("Waiting for the send rate limiter...")

	if err := sendLimiter.Wait(ctx); err != nil {
		WSDebug("Send rate limiter timed out.")
		return errors.Wrap(err, "SendLimiter failed")
	}

	WSDebug("Send has passed the rate limiting.")

	return conn.Send(ctx, b)
}

// Close closes the websocket connection. It assumes that the Websocket is
// closed even when it returns an error. If the Websocket was already closed
// before, ErrWebsocketClosed will be returned.
func (ws *Websocket) Close() error {
	WSDebug("Conn: Acquiring mutex lock to close...")

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	WSDebug("Conn: Write mutex acquired")

	return ws.conn.Close(false)
}

// CloseGracefully is similar to Close, but a proper close frame is sent to
// Discord, invalidating the internal session ID and voiding resumes.
func (ws *Websocket) CloseGracefully() error {
	WSDebug("Conn: Acquiring mutex lock to close...")

	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	WSDebug("Conn: Write mutex acquired")

	return ws.conn.Close(true)
}
