// Package wsutil provides abstractions around the Websocket, including rate
// limits.
package wsutil

import (
	"context"
	"net/url"
	"time"

	"github.com/diamondburned/arikawa/internal/json"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

const DefaultTimeout = time.Minute

type Event struct {
	Data []byte

	// Error is non-nil if Data is nil.
	Error error
}

type Websocket struct {
	Conn Connection
	Addr string

	SendLimiter *rate.Limiter
	DialLimiter *rate.Limiter
}

func New(addr string) *Websocket {
	return NewCustom(NewConn(json.Default{}), addr)
}

// NewCustom creates a new undialed Websocket.
func NewCustom(conn Connection, addr string) *Websocket {
	return &Websocket{
		Conn: conn,
		Addr: addr,

		SendLimiter: NewSendLimiter(),
		DialLimiter: NewDialLimiter(),
	}
}

func (ws *Websocket) Dial(ctx context.Context) error {
	if err := ws.DialLimiter.Wait(ctx); err != nil {
		// Expired, fatal error
		return errors.Wrap(err, "Failed to wait")
	}

	if err := ws.Conn.Dial(ctx, ws.Addr); err != nil {
		return errors.Wrap(err, "Failed to dial")
	}

	// Reset the SendLimiter:
	ws.SendLimiter = NewSendLimiter()

	return nil
}

func (ws *Websocket) Listen() <-chan Event {
	return ws.Conn.Listen()
}

func (ws *Websocket) Send(b []byte) error {
	if err := ws.SendLimiter.Wait(context.Background()); err != nil {
		return errors.Wrap(err, "SendLimiter failed")
	}

	return ws.Conn.Send(b)
}

func (ws *Websocket) Close() error {
	return ws.Conn.Close(websocket.CloseGoingAway)
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
