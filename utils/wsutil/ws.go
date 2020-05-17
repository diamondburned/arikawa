// Package wsutil provides abstractions around the Websocket, including rate
// limits.
package wsutil

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

var (
	// WSTimeout is the timeout for connecting and writing to the Websocket,
	// before Gateway cancels and fails.
	WSTimeout = time.Minute
	// WSBuffer is the size of the Event channel. This has to be at least 1 to
	// make space for the first Event: Ready or Resumed.
	WSBuffer = 10
	// WSError is the default error handler
	WSError = func(err error) { log.Println("Gateway error:", err) }
	// WSExtraReadTimeout is the duration to be added to Hello, as a read
	// timeout for the websocket.
	WSExtraReadTimeout = time.Second
	// WSDebug is used for extra debug logging. This is expected to behave
	// similarly to log.Println().
	WSDebug = func(v ...interface{}) {}
)

type Event struct {
	Data []byte

	// Error is non-nil if Data is nil.
	Error error
}

type Websocket struct {
	Conn Connection
	Addr string

	// Timeout for connecting and writing to the Websocket, uses default
	// WSTimeout (global).
	Timeout time.Duration

	SendLimiter *rate.Limiter
	DialLimiter *rate.Limiter
}

func New(addr string) *Websocket {
	return NewCustom(NewConn(), addr)
}

// NewCustom creates a new undialed Websocket.
func NewCustom(conn Connection, addr string) *Websocket {
	return &Websocket{
		Conn: conn,
		Addr: addr,

		Timeout: WSTimeout,

		SendLimiter: NewSendLimiter(),
		DialLimiter: NewDialLimiter(),
	}
}

func (ws *Websocket) Dial(ctx context.Context) error {
	if ws.Timeout > 0 {
		tctx, cancel := context.WithTimeout(ctx, ws.Timeout)
		defer cancel()

		ctx = tctx
	}

	if err := ws.DialLimiter.Wait(ctx); err != nil {
		// Expired, fatal error
		return errors.Wrap(err, "failed to wait")
	}

	if err := ws.Conn.Dial(ctx, ws.Addr); err != nil {
		return errors.Wrap(err, "failed to dial")
	}

	// Reset the SendLimiter:
	ws.SendLimiter = NewSendLimiter()

	return nil
}

func (ws *Websocket) Listen() <-chan Event {
	return ws.Conn.Listen()
}

func (ws *Websocket) Send(b []byte) error {
	return ws.SendContext(context.Background(), b)
}

// SendContext is a beta API.
func (ws *Websocket) SendContext(ctx context.Context, b []byte) error {
	if err := ws.SendLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "SendLimiter failed")
	}

	return ws.Conn.Send(ctx, b)
}

func (ws *Websocket) Close() error {
	return ws.Conn.Close()
}

func InjectValues(rawurl string, values url.Values) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		// Unknown URL, return as-is.
		return rawurl
	}

	// Append additional parameters:
	var q = u.Query()
	for k, v := range values {
		q[k] = append(q[k], v...)
	}

	u.RawQuery = q.Encode()
	return u.String()
}
