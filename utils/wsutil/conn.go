package wsutil

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// CopyBufferSize is used for the initial size of the internal WS' buffer. Its
// size is 4KB.
var CopyBufferSize = 4096

// MaxCapUntilReset determines the maximum capacity before the bytes buffer is
// re-allocated. It is roughly 16KB, quadruple CopyBufferSize.
var MaxCapUntilReset = CopyBufferSize * 4

// CloseDeadline controls the deadline to wait for sending the Close frame.
var CloseDeadline = time.Second

// ErrWebsocketClosed is returned if the websocket is already closed.
var ErrWebsocketClosed = errors.New("websocket is closed")

// Connection is an interface that abstracts around a generic Websocket driver.
// This connection expects the driver to handle compression by itself, including
// modifying the connection URL. The implementation doesn't have to be safe for
// concurrent use.
type Connection interface {
	// Dial dials the address (string). Context needs to be passed in for
	// timeout. This method should also be re-usable after Close is called.
	Dial(context.Context, string) error

	// Listen returns an event channel that sends over events constantly. It can
	// return nil if there isn't an ongoing connection.
	Listen() <-chan Event

	// Send allows the caller to send bytes. It does not need to clean itself
	// up on errors, as the Websocket wrapper will do that.
	//
	// If the data is nil, it should send a close frame
	Send(context.Context, []byte) error

	// Close should close the websocket connection. The underlying connection
	// may be reused, but this Connection instance will be reused with Dial. The
	// Connection must still be reusable even if Close returns an error.
	Close() error
	// CloseGracefully sends a close frame and then closes the websocket
	// connection.
	CloseGracefully() error
}

// Conn is the default Websocket connection. It tries to compresses all payloads
// using zlib.
type Conn struct {
	Dialer websocket.Dialer
	Header http.Header
	Conn   *websocket.Conn
	events chan Event
}

var _ Connection = (*Conn)(nil)

// NewConn creates a new default websocket connection with a default dialer.
func NewConn() *Conn {
	return NewConnWithDialer(websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  WSTimeout,
		ReadBufferSize:    CopyBufferSize,
		WriteBufferSize:   CopyBufferSize,
		EnableCompression: true,
	})
}

// NewConnWithDialer creates a new default websocket connection with a custom
// dialer.
func NewConnWithDialer(dialer websocket.Dialer) *Conn {
	return &Conn{
		Dialer: dialer,
		Header: http.Header{
			"Accept-Encoding": {"zlib"},
		},
	}
}

func (c *Conn) Dial(ctx context.Context, addr string) (err error) {
	// BUG which prevents stream compression.
	// See https://github.com/golang/go/issues/31514.

	c.Conn, _, err = c.Dialer.DialContext(ctx, addr, c.Header)
	if err != nil {
		return errors.Wrap(err, "failed to dial WS")
	}

	// Reset the deadline.
	c.Conn.SetWriteDeadline(resetDeadline)

	c.events = make(chan Event, WSBuffer)
	go startReadLoop(c.Conn, c.events)

	return err
}

// Listen returns an event channel if there is a connection associated with it.
// It returns nil if there is none.
func (c *Conn) Listen() <-chan Event {
	return c.events
}

// resetDeadline is used to reset the write deadline after using the context's.
var resetDeadline = time.Time{}

func (c *Conn) Send(ctx context.Context, b []byte) error {
	d, ok := ctx.Deadline()
	if ok {
		c.Conn.SetWriteDeadline(d)
		defer c.Conn.SetWriteDeadline(resetDeadline)
	}

	if err := c.Conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return err
	}

	return nil
}

func (c *Conn) Close() error {
	WSDebug("Conn: Close is called; shutting down the Websocket connection.")

	// Have a deadline before closing.
	var deadline = time.Now().Add(5 * time.Second)
	c.Conn.SetWriteDeadline(deadline)

	// Close the WS.
	err := c.Conn.Close()

	c.Conn.SetWriteDeadline(resetDeadline)

	WSDebug("Conn: Websocket closed; error:", err)
	WSDebug("Conn: Flushing events...")

	// Flush all events before closing the channel. This will return as soon as
	// c.events is closed, or after closed.
	for range c.events {
	}

	WSDebug("Flushed events.")

	return err
}

func (c *Conn) CloseGracefully() error {
	WSDebug("Conn: CloseGracefully is called; sending close frame.")

	c.Conn.SetWriteDeadline(time.Now().Add(CloseDeadline))

	err := c.Conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		WSError(err)
	}

	WSDebug("Conn: Close frame sent; error:", err)

	return c.Close()
}

// loopState is a thread-unsafe disposable state container for the read loop.
// It's made to completely separate the read loop of any synchronization that
// doesn't involve the websocket connection itself.
type loopState struct {
	conn *websocket.Conn
	zlib io.ReadCloser
	buf  bytes.Buffer
}

func startReadLoop(conn *websocket.Conn, eventCh chan<- Event) {
	// Clean up the events channel in the end.
	defer close(eventCh)

	// Allocate the read loop its own private resources.
	state := loopState{conn: conn}
	state.buf.Grow(CopyBufferSize)

	for {
		b, err := state.handle()
		if err != nil {
			WSDebug("Conn: Read error:", err)

			// Is the error an EOF?
			if errors.Is(err, io.EOF) {
				// Yes it is, exit.
				return
			}

			// Is the error an intentional close call? Go 1.16 exposes
			// ErrClosing, but we have to do this for now.
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}

			// Unusual error; log and exit:
			eventCh <- Event{nil, errors.Wrap(err, "WS error")}
			return
		}

		// If the payload length is 0, skip it.
		if len(b) == 0 {
			continue
		}

		eventCh <- Event{b, nil}
	}
}

func (state *loopState) handle() ([]byte, error) {
	// skip message type
	t, r, err := state.conn.NextReader()
	if err != nil {
		return nil, err
	}

	if t == websocket.BinaryMessage {
		// Probably a zlib payload.

		if state.zlib == nil {
			z, err := zlib.NewReader(r)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create a zlib reader")
			}
			state.zlib = z
		} else {
			if err := state.zlib.(zlib.Resetter).Reset(r, nil); err != nil {
				return nil, errors.Wrap(err, "failed to reset zlib reader")
			}
		}

		defer state.zlib.Close()
		r = state.zlib
	}

	return state.readAll(r)
}

// readAll reads bytes into an existing buffer, copy it over, then wipe the old
// buffer.
func (state *loopState) readAll(r io.Reader) ([]byte, error) {
	defer state.buf.Reset()

	if _, err := state.buf.ReadFrom(r); err != nil {
		return nil, err
	}

	// Copy the bytes so we could empty the buffer for reuse.
	cpy := make([]byte, state.buf.Len())
	copy(cpy, state.buf.Bytes())

	// If the buffer's capacity is over the limit, then re-allocate a new one.
	if state.buf.Cap() > MaxCapUntilReset {
		state.buf = bytes.Buffer{}
		state.buf.Grow(CopyBufferSize)
	}

	return cpy, nil
}
