package wsutil

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	stderr "errors"

	"github.com/diamondburned/arikawa/internal/json"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const CopyBufferSize = 2048

// CloseDeadline controls the deadline to wait for sending the Close frame.
var CloseDeadline = time.Second

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
	Send([]byte) error

	// Close should close the websocket connection. The connection will not be
	// reused. Code should be sent as the status code for the close frame.
	Close(code int) error
}

// Conn is the default Websocket connection. It compresses all payloads using
// zlib.
type Conn struct {
	Conn *websocket.Conn
	json.Driver

	dialer *websocket.Dialer
	mut    sync.RWMutex
	events chan Event

	buf bytes.Buffer

	// zlib *zlib.Inflator // zlib.NewReader
	// buf  []byte         // io.Copy buffer
}

var _ Connection = (*Conn)(nil)

func NewConn(driver json.Driver) *Conn {
	return &Conn{
		Driver: driver,
		dialer: &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  DefaultTimeout,
			EnableCompression: true,
		},
		events: make(chan Event),
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

	c.mut.Lock()
	defer c.mut.Unlock()

	c.Conn, _, err = c.dialer.DialContext(ctx, addr, headers)
	if err != nil {
		return errors.Wrap(err, "Failed to dial WS")
	}

	c.events = make(chan Event)
	go c.readLoop()
	return err
}

func (c *Conn) Listen() <-chan Event {
	return c.events
}

func (c *Conn) readLoop() {
	// Acquire the read lock throughout the span of the loop. This would still
	// allow Send to acquire another RLock, but wouldn't allow Close to
	// prematurely exit, as Close acquires a write lock.
	c.mut.RLock()
	defer c.mut.RUnlock()

	// Clean up the events channel in the end.
	defer close(c.events)

	for {
		b, err := c.handle()
		if err != nil {
			// Is the error an EOF?
			if stderr.Is(err, io.EOF) {
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

		// If nil bytes, then it's an incomplete payload.
		if b == nil {
			continue
		}

		c.events <- Event{b, nil}
	}
}

func (c *Conn) handle() ([]byte, error) {
	// skip message type
	t, r, err := c.Conn.NextReader()
	if err != nil {
		return nil, err
	}

	if t == websocket.BinaryMessage {
		// Probably a zlib payload
		z, err := zlib.NewReader(r)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create a zlib reader")
		}

		defer z.Close()
		r = z
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

func (c *Conn) Send(b []byte) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	if c.Conn == nil {
		return errors.New("Websocket is closed.")
	}

	return c.Conn.WriteMessage(websocket.TextMessage, b)
}

func (c *Conn) Close(code int) error {
	// Wait for the read loop to exit at the end.
	err := c.writeClose(code)
	c.close()
	return err
}

func (c *Conn) writeClose(code int) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	// Quick deadline:
	deadline := time.Now().Add(CloseDeadline)

	// Make a closure message:
	msg := websocket.FormatCloseMessage(code, "")

	// Send a close message before closing the connection. We're not error
	// checking this because it's not important.
	c.Conn.WriteControl(websocket.TextMessage, msg, deadline)

	// Safe to close now.
	return c.Conn.Close()
}

func (c *Conn) close() {
	// Flush all events:
	for range c.events {
	}

	// This blocks until the events channel is dead.
	c.mut.Lock()
	defer c.mut.Unlock()

	// Clean up.
	c.events = nil
	c.Conn = nil
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
