// Package wsutil provides abstractions around the Websocket, including rate
// limits.
package wsutil

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/internal/json"
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

	listener <-chan Event
	dialed   bool
}

func New(addr string) (*Websocket, error) {
	return NewCustom(NewConn(json.Default{}), addr)
}

// NewCustom creates a new undialed Websocket.
func NewCustom(conn Connection, addr string) (*Websocket, error) {
	ws := &Websocket{
		Conn: conn,
		Addr: addr,

		SendLimiter: NewSendLimiter(),
		DialLimiter: NewDialLimiter(),
	}

	return ws, nil
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

func (ws *Websocket) Send(ctx context.Context, b []byte) error {
	if err := ws.SendLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "SendLimiter failed")
	}

	return ws.Conn.Send(ctx, b)
}

func (ws *Websocket) Close(err error) error {
	return ws.Conn.Close(err)
}
