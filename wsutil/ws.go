package wsutil

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/json"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

const DefaultTimeout = 10 * time.Second

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
}

func New(ctx context.Context, addr string) (*Websocket, error) {
	return NewCustom(ctx, NewConn(json.Default{}), addr)
}

// NewCustom creates a new undialed Websocket.
func NewCustom(
	ctx context.Context, conn Connection, addr string) (*Websocket, error) {

	ws := &Websocket{
		Conn: conn,
		Addr: addr,

		SendLimiter: NewSendLimiter(),
		DialLimiter: NewDialLimiter(),
	}

	return ws, nil
}

func (ws *Websocket) Redial(ctx context.Context) error {
	if err := ws.DialLimiter.Wait(ctx); err != nil {
		// Expired, fatal error
		return errors.Wrap(err, "Failed to wait")
	}

	if err := ws.Conn.Dial(ctx, ws.Addr); err != nil {
		return errors.Wrap(err, "Failed to dial")
	}

	return nil
}

func (ws *Websocket) Listen() <-chan Event {
	if ws.listener == nil {
		ws.listener = ws.Conn.Listen()
	}
	return ws.listener
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
