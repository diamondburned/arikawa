package udp

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type pauseSignals struct {
	ctx    context.Context
	cancel func()
	done   chan struct{}
}

func newPauseSignals() *pauseSignals {
	ctx, cancel := context.WithCancel(context.Background())

	return &pauseSignals{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// ErrManagerClosed is returned when a Manager that is already closed is dialed,
// written to or read from.
var ErrManagerClosed = errors.New("manager is closed")

// ErrDialWhileUnpaused is returned if Dial is called on the Manager without
// pausing it first
var ErrDialWhileUnpaused = errors.New("dial is called while manager is not paused")

// Manager manages a UDP connection. It allows reconnecting. A Manager instance
// is thread-safe, meaning it can be used concurrently.
type Manager struct {
	mu     sync.Mutex
	closed chan struct{} // will perma-close, should never use blocking
	dialer *net.Dialer

	// paused is not nil if we're paused
	paused *pauseSignals
	// conn state, can be nil
	conn *Connection

	frequency time.Duration
	timeIncr  uint32
}

// NewManager creates a new UDP connection manager.
func NewManager() *Manager {
	return NewManagerWithDialer(&Dialer)
}

// NewManagerWithDialer creates a new UDP connection manager with a custom
// dialer.
func NewManagerWithDialer(d *net.Dialer) *Manager {
	return &Manager{
		closed: make(chan struct{}),
		dialer: d,
	}
}

// SetDialer sets the manager's dialer.
func (m *Manager) SetDialer(d *net.Dialer) {
	m.mu.Lock()
	m.dialer = d
	m.mu.Unlock()
}

// Close closes the current connection. If the connection is already closed,
// then nothing is done.
func (m *Manager) Close() (err error) {
	m.mu.Lock()

	// Ensure we don't close this channel twice.
	select {
	case <-m.closed:
	default:
		close(m.closed)
	}

	if m.paused != nil {
		m.paused.cancel()
		close(m.paused.done) // unpause directly
		m.paused = nil
	}

	if m.conn != nil {
		err = m.conn.Close()
		m.conn = nil
	}

	m.mu.Unlock()
	return
}

// Pause closes the current connection and pauses the current manager. It halts
// other users of the manager until Unpause is called.
func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}

	if m.paused == nil {
		m.paused = newPauseSignals()
	}
}

// Unpause unpauses the current manager.
func (m *Manager) Unpause() {
	log.Println("acquiring mutex")
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.paused != nil {
		m.paused.cancel()
		log.Println("unpaused closing")
		close(m.paused.done)
		m.paused = nil
	}

	log.Println("unpaused")
}

// Dial dials the internal connection to the given address and SSRC number. The
// caller must call Dial after Pause and before Unpause. If the Manager is
// already being dialed elsewhere, then ErrAlreadyDialing is returned.
func (m *Manager) Dial(addr string, ssrc uint32) (*Connection, error) {
	m.mu.Lock()
	if m.paused == nil {
		m.mu.Unlock()
		return nil, ErrDialWhileUnpaused
	}

	// Reinitialize the close channel.
	m.closed = make(chan struct{})

	dialer := m.dialer
	signals := m.paused
	m.mu.Unlock()

	conn, err := DialConnectionCustom(signals.ctx, dialer, addr, ssrc)
	if err != nil {
		m.doneReconnecting(signals, nil)
		return nil, errors.Wrap(err, "failed to dial")
	}

	if !m.doneReconnecting(signals, conn) {
		return nil, ErrManagerClosed
	}

	return conn, nil
}

func (m *Manager) doneReconnecting(signals *pauseSignals, conn *Connection) bool {
	if conn == nil {
		return false
	}

	// If we have successfully reconnected, then acquire the lock, set the
	// connection, and then signal.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Recheck if the manager is closed, since that might happen while we
	// weren't acquiring the mutex.
	select {
	case <-m.closed:
		// Manager is closed; discard the new connection.
		conn.Close()
		return false
	default:
		// ok
	}

	m.conn = conn
	m.paused = nil

	if m.frequency > 0 && m.timeIncr > 0 {
		conn.ResetFrequency(m.frequency, m.timeIncr)
	}

	return true
}

// ResetFrequency sets the current connection and future connections' write
// frequency. Note that calling this method while Connection is being used in a
// different goroutine is not thread-safe.
func (m *Manager) ResetFrequency(frameDuration time.Duration, timeIncr uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.frequency = frameDuration
	m.timeIncr = timeIncr

	if m.conn != nil {
		m.conn.ResetFrequency(frameDuration, timeIncr)
	}
}

// ReadPacket reads the current packet. It blocks until a packet arrives or
// the Manager is closed.
func (m *Manager) ReadPacket() (p *Packet, err error) {
	err = m.acquire(func(conn *Connection) (err error) {
		p, err = conn.ReadPacket()
		return
	})
	return
}

// Write writes to the current connection in the manager. It blocks if the
// connection is being re-established.
func (m *Manager) Write(b []byte) (n int, err error) {
	err = m.acquire(func(conn *Connection) (err error) {
		log.Printf("acquired, writing to ptr %p", conn)
		n, err = conn.Write(b)
		return
	})
	return
}

func (m *Manager) acquire(f func(conn *Connection) error) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	var err error
	for {
		if conn != nil {
			if err = f(conn); err == nil {
				return nil
			}

			// Investigate why we've failed. If the connection is closed, then
			// net.ErrClosed is returned, and it probably means that we're
			// reconnecting, otherwise the m.closed channel will get us
			// something. If the error isn't, then it's an unexpected error, so
			// we bail.
			if !errors.Is(err, net.ErrClosed) {
				return err
			}
		}

		m.mu.Lock()

		// If connection was nil but is no longer nil k when we didn't have the
		// mutex, then use that connection.
		if conn == nil && m.conn != nil {
			conn = m.conn
			m.mu.Unlock()
			continue
		}

		// Take the current signals and unlock the mutex. This avoids deadlocks
		// with Dial.
		paused := m.paused
		closing := m.closed
		m.mu.Unlock()

		// Something is going wrong if we're not reconnecting: the connection is
		// probably closed. Otherwise, Dial would've filled it up.
		if paused == nil {
			return ErrManagerClosed
		}

		// Wait until either we've reconnected or the manager is closed. This
		// might run several times before conn is actually set, since we might
		// be stealing Dial's mutex.
		select {
		case <-closing:
			return ErrManagerClosed
		case <-paused.done:
			// We've reconnected. Reacquire the connection then retry.
			m.mu.Lock()
			conn = m.conn
			m.mu.Unlock()
		}
	}
}
