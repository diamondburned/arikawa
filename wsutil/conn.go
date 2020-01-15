package wsutil

import (
	"compress/zlib"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/json"
	"github.com/pkg/errors"
	"nhooyr.io/websocket"
)

var WSBuffer = 12
var WSReadLimit = 4096 // 4096 bytes

// Connection is an interface that abstracts around a generic Websocket driver.
// This connection expects the driver to handle compression by itself.
type Connection interface {
	// Dial dials the address (string). Context needs to be passed in for
	// timeout. This method should also be re-usable after Close is called.
	Dial(context.Context, string) error

	// Listen sends over events constantly. Error will be non-nil if Data is
	// nil, so check for Error first.
	Listen() <-chan Event

	// Send allows the caller to send bytes. Context needs to be passed in order
	// to re-use the context that's already used for the limiter.
	Send(context.Context, []byte) error

	// Close should close the websocket connection. The connection will not be
	// reused.
	// If error is nil, the connection should close with a StatusNormalClosure
	// (1000). If not, it should close with a StatusProtocolError (1002).
	Close(err error) error
}

// Conn is the default Websocket connection. It compresses all payloads using
// zlib.
type Conn struct {
	*websocket.Conn
	json.Driver

	ReadTimeout time.Duration // DefaultTimeout

	events chan Event
}

var _ Connection = (*Conn)(nil)

func NewConn(driver json.Driver) *Conn {
	return &Conn{
		Driver:      driver,
		ReadTimeout: DefaultTimeout,
	}
}

func (c *Conn) Dial(ctx context.Context, addr string) error {
	var err error

	headers := http.Header{}
	headers.Set("Accept-Encoding", "zlib") // enable

	c.Conn, _, err = websocket.Dial(ctx, addr, &websocket.DialOptions{
		HTTPHeader: headers,
	})

	return err
}

func (c *Conn) Listen() <-chan Event {
	c.events = make(chan Event, WSBuffer)
	go func() { c.readLoop(c.events) }()
	return c.events
}

func (c *Conn) readLoop(ch chan Event) {
	defer close(ch)

	for {
		ctx, cancel := context.WithTimeout(
			context.Background(), c.ReadTimeout)
		defer cancel()

		b, err := c.readAll(ctx)
		if err != nil {
			ch <- Event{nil, errors.Wrap(err, "WS error")}

			// Check if the error is a fatal one
			if websocket.CloseStatus(err) > -1 {
				// Error is fatal, exit
				return
			}

			// or it's not fatal, we just continue
			continue
		}

		ch <- Event{b, nil}
	}
}

func (c *Conn) readAll(ctx context.Context) ([]byte, error) {
	t, r, err := c.Reader(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "WS error")
	}

	if t == websocket.MessageBinary {
		// Probably a zlib payload
		z, err := zlib.NewReader(r)
		if err != nil {
			return nil,
				errors.Wrap(err, "Failed to create a zlib reader")
		}

		defer z.Close()
		r = z
	}

	return ioutil.ReadAll(r)
}

func (c *Conn) Send(ctx context.Context, b []byte) error {
	w, err := c.Writer(ctx, websocket.MessageBinary)
	if err != nil {
		return errors.Wrap(err, "Failed to get WS writer")
	}

	defer w.Close()

	// Compress with zlib by default.
	w = zlib.NewWriter(w)

	_, err = w.Write(b)
	return err
}

func (c *Conn) Close(err error) error {
	// Close the event channels
	defer close(c.events)

	if err == nil {
		return c.Conn.Close(websocket.StatusNormalClosure, "")
	}

	var msg = err.Error()
	if len(msg) > 125 {
		msg = msg[:125] // truncate
	}

	return c.Conn.Close(websocket.StatusProtocolError, msg)
}
