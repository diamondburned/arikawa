package wsutil

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/utils/json"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const CopyBufferSize = 2048

// CloseDeadline controls the deadline to wait for sending the Close frame.
var CloseDeadline = time.Second

// ErrWebsocketClosed is returned if the websocket is already closed.
var ErrWebsocketClosed = errors.New("websocket is closed")

// Connection is an interface that abstracts around a generic Websocket driver.
// This connection expects the driver to handle compression by itself, including
// modifying the connection URL.
type Connection interface {
	// Dial dials the address (string). Context needs to be passed in for
	// timeout. This method should also be re-usable after Close is called.
	Dial(context.Context, string) error

	// Listen sends over events constantly. Error will be non-nil if Data is
	// nil, so check for Error first.
	Listen() <-chan Event

	// Send allows the caller to send bytes. Thread safety is a requirement.
	Send(context.Context, []byte) error

	// Close should close the websocket connection. The connection will not be
	// reused.
	Close() error
}

// Conn is the default Websocket connection. It compresses all payloads using
// zlib.
type Conn struct {
	Conn *websocket.Conn
	json.Driver

	dialer *websocket.Dialer
	events chan Event

	// write channels
	writes chan []byte
	errors chan error

	buf  bytes.Buffer
	zlib io.ReadCloser // (compress/zlib).reader

	// nil until Dial().
	closeOnce *sync.Once

	// zlib *zlib.Inflator // zlib.NewReader
	// buf  []byte         // io.Copy buffer
}

var _ Connection = (*Conn)(nil)

func NewConn() *Conn {
	return NewConnWithDriver(json.Default)
}

func NewConnWithDriver(driver json.Driver) *Conn {
	return &Conn{
		Driver: driver,
		dialer: &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  WSTimeout,
			EnableCompression: true,
		},
		// zlib:   zlib.NewInflator(),
		// buf:    make([]byte, CopyBufferSize),
	}
}

func (c *Conn) Dial(ctx context.Context, addr string) error {
	var err error

	// Enable compression:
	headers := http.Header{}
	headers.Set("Accept-Encoding", "zlib")

	// BUG: https://github.com/golang/go/issues/31514
	// // Enable stream compression:
	// addr = InjectValues(addr, url.Values{
	// 	"compress": {"zlib-stream"},
	// })

	c.Conn, _, err = c.dialer.DialContext(ctx, addr, headers)
	if err != nil {
		return errors.Wrap(err, "failed to dial WS")
	}

	// Set up the closer.
	c.closeOnce = &sync.Once{}

	c.events = make(chan Event)
	go c.readLoop()

	c.writes = make(chan []byte)
	c.errors = make(chan error)
	go c.writeLoop()

	return err
}

func (c *Conn) Listen() <-chan Event {
	return c.events
}

func (c *Conn) readLoop() {
	// Acquire the read lock throughout the span of the loop. This would still
	// allow Send to acquire another RLock, but wouldn't allow Close to
	// prematurely exit, as Close acquires a write lock.
	// c.mut.RLock()
	// defer c.mut.RUnlock()

	// Clean up the events channel in the end.
	defer close(c.events)

	for {
		b, err := c.handle()
		if err != nil {
			// Is the error an EOF?
			if errors.Is(err, io.EOF) {
				// Yes it is, exit.
				return
			}

			// Check if the error is a normal one:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}

			// Unusual error; log and exit:
			c.events <- Event{nil, errors.Wrap(err, "WS error")}
			return
		}

		// If the payload length is 0, skip it.
		if len(b) == 0 {
			continue
		}

		c.events <- Event{b, nil}
	}
}

func (c *Conn) writeLoop() {
	// Closig c.writes would break the loop immediately.
	for bytes := range c.writes {
		c.errors <- c.Conn.WriteMessage(websocket.TextMessage, bytes)
	}

	// Quick deadline:
	deadline := time.Now().Add(CloseDeadline)

	// Make a closure message:
	msg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "")

	// Send a close message before closing the connection. We're not error
	// checking this because it's not important.
	c.Conn.WriteControl(websocket.TextMessage, msg, deadline)

	// Safe to close now.
	c.errors <- c.Conn.Close()
	close(c.errors)
}

func (c *Conn) handle() ([]byte, error) {
	// skip message type
	t, r, err := c.Conn.NextReader()
	if err != nil {
		return nil, err
	}

	if t == websocket.BinaryMessage {
		// Probably a zlib payload

		if c.zlib == nil {
			z, err := zlib.NewReader(r)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create a zlib reader")
			}
			c.zlib = z
		} else {
			if err := c.zlib.(zlib.Resetter).Reset(r, nil); err != nil {
				return nil, errors.Wrap(err, "failed to reset zlib reader")
			}
		}

		defer c.zlib.Close()
		r = c.zlib
	}

	return readAll(&c.buf, r)

	// if t is a text message, then handle it normally.
	// if t == websocket.TextMessage {
	// 	return readAll(&c.buf, r)
	// }

	// // Write to the zlib writer.
	// c.zlib.Write(r)
	// // if _, err := io.CopyBuffer(c.zlib, r, c.buf); err != nil {
	// // 	return nil, errors.Wrap(err, "Failed to write to zlib")
	// // }

	// if !c.zlib.CanFlush() {
	// 	return nil, nil
	// }

	// // Flush and get the uncompressed payload.
	// b, err := c.zlib.Flush()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "Failed to flush zlib")
	// }

	// return nil, errors.New("Unexpected binary message.")
}

func (c *Conn) Send(ctx context.Context, b []byte) error {
	// If websocket is already closed.
	if c.writes == nil {
		return ErrWebsocketClosed
	}

	// Send the bytes.
	select {
	case c.writes <- b:
		// continue
	case <-ctx.Done():
		return ctx.Err()
	}

	// Receive the error.
	select {
	case err := <-c.errors:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) Close() (err error) {
	// Use a sync.Once to guarantee that other Close() calls block until the
	// main call is done. It also prevents future calls.
	c.closeOnce.Do(func() {
		// Close c.writes. This should trigger the websocket to close itself.
		close(c.writes)
		// Mark c.writes as empty.
		c.writes = nil

		// Wait for the write loop to exit by flusing the errors channel.
		err = <-c.errors     // get close error
		for range c.errors { // then flush
		}

		// Flush all events before closing the channel. This will return as soon as
		// c.events is closed, or after closed.
		for range c.events {
		}

		// Mark c.events as empty.
		c.events = nil

		// Mark c.Conn as empty.
		c.Conn = nil
	})

	return err
}

// readAll reads bytes into an existing buffer, copy it over, then wipe the old
// buffer.
func readAll(buf *bytes.Buffer, r io.Reader) ([]byte, error) {
	defer buf.Reset()
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}

	// Copy the bytes so we could empty the buffer for reuse.
	p := buf.Bytes()
	cpy := make([]byte, len(p))
	copy(cpy, p)

	return cpy, nil
}
