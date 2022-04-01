package ws

import (
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const rwBufferSize = 1 << 15 // 32KB

// ErrWebsocketClosed is returned if the websocket is already closed.
var ErrWebsocketClosed = errors.New("websocket is closed")

// Connection is an interface that abstracts around a generic Websocket driver.
// This connection expects the driver to handle compression by itself, including
// modifying the connection URL. The implementation doesn't have to be safe for
// concurrent use.
type Connection interface {
	// Dial dials the address (string). Context needs to be passed in for
	// timeout. This method should also be re-usable after Close is called.
	Dial(context.Context, string) (<-chan Op, error)

	// Send allows the caller to send bytes.
	Send(context.Context, []byte) error

	// Close should close the websocket connection. The underlying connection
	// may be reused, but this Connection instance will be reused with Dial. The
	// Connection must still be reusable even if Close returns an error. If
	// gracefully is true, then the implementation must send a close frame
	// prior.
	Close(gracefully bool) error
}

// Conn is the default Websocket connection. It tries to compresses all payloads
// using zlib.
type Conn struct {
	dialer websocket.Dialer
	codec  Codec

	// conn is used for synchronizing the conn instance itself. Any use of conn
	// must copy conn out.
	conn *connMutex
	// mut is used for synchronizing the conn field.
	mut sync.Mutex

	// CloseTimeout is the timeout for graceful closing. It's defaulted to 5s.
	CloseTimeout time.Duration
}

type connMutex struct {
	*websocket.Conn
	wrmut  chan struct{}
	cancel context.CancelFunc
}

var _ Connection = (*Conn)(nil)

// NewConn creates a new default websocket connection with a default dialer.
func NewConn(codec Codec) *Conn {
	return NewConnWithDialer(codec, websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		ReadBufferSize:    rwBufferSize,
		WriteBufferSize:   rwBufferSize,
		EnableCompression: true,
	})
}

// NewConnWithDialer creates a new default websocket connection with a custom
// dialer.
func NewConnWithDialer(codec Codec, dialer websocket.Dialer) *Conn {
	return &Conn{
		dialer:       dialer,
		codec:        codec,
		CloseTimeout: 5 * time.Second,
	}
}

// Dial starts a new connection and returns the listening channel for it. If the
// websocket is already dialed, then the connection is closed first.
func (c *Conn) Dial(ctx context.Context, addr string) (<-chan Op, error) {
	// BUG which prevents stream compression.
	// See https://github.com/golang/go/issues/31514.

	c.mut.Lock()
	defer c.mut.Unlock()

	// Ensure that the connection is already closed.
	if c.conn != nil {
		c.conn.close(c.CloseTimeout, false)
	}

	conn, _, err := c.dialer.DialContext(ctx, addr, c.codec.Headers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial WS")
	}

	ctx, cancel := context.WithCancel(context.Background())

	events := make(chan Op, 1)
	go readLoop(ctx, conn, c.codec, events)

	c.conn = &connMutex{
		wrmut:  make(chan struct{}, 1),
		Conn:   conn,
		cancel: cancel,
	}

	return events, err
}

// Close implements Connection.
func (c *Conn) Close(gracefully bool) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.conn.close(c.CloseTimeout, gracefully)
}

func (c *connMutex) close(timeout time.Duration, gracefully bool) error {
	if c == nil || c.Conn == nil {
		WSDebug("Conn: Close is called on already closed connection")
		return ErrWebsocketClosed
	}

	WSDebug("Conn: Close is called; shutting down the Websocket connection.")

	if gracefully {
		// Have a deadline before closing.
		deadline := time.Now().Add(timeout)

		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		select {
		case c.wrmut <- struct{}{}:
			// Lock acquired. We can now safely set the deadline and write.
			c.SetWriteDeadline(deadline)

			WSDebug("Conn: Graceful closing requested, sending close frame.")

			if err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			); err != nil {
				WSError(err)
			}

			// Release the lock.
			<-c.wrmut

		case <-ctx.Done():
			// We couldn't acquire the lock. Resort to just closing the
			// connection directly.
		}
	}

	// Close the WS.
	err := c.Conn.Close()

	if err != nil {
		WSDebug("Conn: Websocket closed; error:", err)
	} else {
		WSDebug("Conn: Websocket closed successfully")
	}

	c.Conn = nil

	c.cancel()
	c.cancel = nil

	return err
}

// resetDeadline is used to reset the write deadline after using the context's.
var resetDeadline = time.Time{}

// Send implements Connection.
func (c *Conn) Send(ctx context.Context, b []byte) error {
	c.mut.Lock()
	conn := c.conn
	c.mut.Unlock()

	if conn == nil || conn.Conn == nil {
		return ErrWebsocketClosed
	}

	select {
	case conn.wrmut <- struct{}{}:
		defer func() { <-conn.wrmut }()

		if ctx != context.Background() {
			d, ok := ctx.Deadline()
			if ok {
				conn.SetWriteDeadline(d)
				defer conn.SetWriteDeadline(resetDeadline)
			}
		}

		return conn.WriteMessage(websocket.TextMessage, b)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// loopState is a thread-unsafe disposable state container for the read loop.
// It's made to completely separate the read loop of any synchronization that
// doesn't involve the websocket connection itself.
type loopState struct {
	conn  *websocket.Conn
	codec Codec
	zlib  io.ReadCloser
	buf   DecodeBuffer
}

func readLoop(ctx context.Context, conn *websocket.Conn, codec Codec, opCh chan<- Op) {
	// Clean up the events channel in the end.
	defer close(opCh)

	// Allocate the read loop its own private resources.
	state := loopState{
		conn:  conn,
		codec: codec,
		buf:   NewDecodeBuffer(1 << 14), // 16KB
	}

	for {
		if err := state.handle(ctx, opCh); err != nil {
			WSDebug("Conn: fatal Conn error:", err)

			closeEv := &CloseEvent{
				Err:  err,
				Code: -1,
			}

			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				closeEv.Code = closeErr.Code
				closeEv.Err = fmt.Errorf("%d %s", closeErr.Code, closeErr.Text)
			}

			opCh <- Op{
				Code: closeEv.Op(),
				Type: closeEv.EventType(),
				Data: closeEv,
			}

			return
		}
	}
}

func (state *loopState) handle(ctx context.Context, opCh chan<- Op) error {
	// skip message type
	t, r, err := state.conn.NextReader()
	if err != nil {
		return err
	}

	if t == websocket.BinaryMessage {
		// Probably a zlib payload.

		if state.zlib == nil {
			z, err := zlib.NewReader(r)
			if err != nil {
				return errors.Wrap(err, "failed to create a zlib reader")
			}
			state.zlib = z
		} else {
			if err := state.zlib.(zlib.Resetter).Reset(r, nil); err != nil {
				return errors.Wrap(err, "failed to reset zlib reader")
			}
		}

		defer state.zlib.Close()
		r = state.zlib
	}

	if err := state.codec.DecodeInto(ctx, r, &state.buf, opCh); err != nil {
		return errors.Wrap(err, "error distributing event")
	}

	return nil
}
