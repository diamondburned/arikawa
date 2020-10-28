package wsutil

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"sync"
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
	mutex sync.Mutex

	Conn *websocket.Conn

	dialer *websocket.Dialer
	events chan Event
}

var _ Connection = (*Conn)(nil)

// NewConn creates a new default websocket connection with a default dialer.
func NewConn() *Conn {
	return NewConnWithDialer(&websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  WSTimeout,
		ReadBufferSize:    CopyBufferSize,
		WriteBufferSize:   CopyBufferSize,
		EnableCompression: true,
	})
}

// NewConn creates a new default websocket connection with a custom dialer.
func NewConnWithDialer(dialer *websocket.Dialer) *Conn {
	return &Conn{dialer: dialer}
}

func (c *Conn) Dial(ctx context.Context, addr string) error {
	// Enable compression:
	headers := http.Header{
		"Accept-Encoding": {"zlib"},
	}

	// BUG which prevents stream compression.
	// See https://github.com/golang/go/issues/31514.

	conn, _, err := c.dialer.DialContext(ctx, addr, headers)
	if err != nil {
		return errors.Wrap(err, "failed to dial WS")
	}

	events := make(chan Event, WSBuffer)
	go startReadLoop(conn, events)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Conn = conn
	c.events = events

	return err
}

func (c *Conn) Listen() <-chan Event {
	return c.events
}

// resetDeadline is used to reset the write deadline after using the context's.
var resetDeadline = time.Time{}

func (c *Conn) Send(ctx context.Context, b []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	d, ok := ctx.Deadline()
	if ok {
		c.Conn.SetWriteDeadline(d)
		defer c.Conn.SetWriteDeadline(resetDeadline)
	}

	return c.Conn.WriteMessage(websocket.TextMessage, b)
}

func (c *Conn) Close() error {
	// Use a sync.Once to guarantee that other Close() calls block until the
	// main call is done. It also prevents future calls.
	WSDebug("Conn: Acquiring write lock...")

	// Acquire the write lock forever.
	c.mutex.Lock()
	defer c.mutex.Unlock()

	WSDebug("Conn: Write lock acquired; closing.")

	// Close the WS.
	err := c.closeWS()

	WSDebug("Conn: Websocket closed; error:", err)
	WSDebug("Conn: Flusing events...")

	// Flush all events before closing the channel. This will return as soon as
	// c.events is closed, or after closed.
	for range c.events {
	}

	WSDebug("Flushed events.")

	// Mark c.Conn as empty.
	c.Conn = nil

	return err
}

func (c *Conn) closeWS() error {
	// We can't close with a write control here, since it will invalidate the
	// old session, breaking resumes.

	// // Quick deadline:
	// deadline := time.Now().Add(CloseDeadline)

	// // Make a closure message:
	// msg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "")

	// // Send a close message before closing the connection. We're not error
	// // checking this because it's not important.
	// err = c.Conn.WriteControl(websocket.CloseMessage, msg, deadline)

	return c.Conn.Close()
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
		// Probably a zlib payload

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
