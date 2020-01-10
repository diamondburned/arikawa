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
	conn Connection

	WriteTimeout time.Duration
	SendLimiter  *rate.Limiter
}

func New(ctx context.Context,
	driver json.Driver, addr string) (*Websocket, error) {

	if driver == nil {
		driver = json.Default{}
	}

	c := NewConn(driver)
	if err := c.Dial(ctx, addr); err != nil {
		return nil, errors.Wrap(err, "Failed to dial")
	}

	return NewWithConn(c, driver), nil
}

// NewWithConn uses an already-dialed connection for Websocket.
func NewWithConn(conn Connection, driver json.Driver) *Websocket {
	return &Websocket{
		conn: conn,

		WriteTimeout: DefaultTimeout,
		SendLimiter:  NewSendLimiter(),
	}
}

func (ws *Websocket) Listen() <-chan Event {
	return ws.conn.Listen()
}

func (ws *Websocket) Send(b []byte) error {
	ctx, cancel := context.WithTimeout(
		context.Background(), ws.WriteTimeout)
	defer cancel()

	if err := ws.SendLimiter.Wait(ctx); err != nil {
		return errors.Wrap(err, "SendLimiter failed")
	}

	return ws.conn.Send(ctx, b)
}
