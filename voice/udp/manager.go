package udp

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// ErrManagerClosed is returned when a Manager that is already closed is dialed,
// written to or read from.
var ErrManagerClosed = errors.New("manager is closed")

// ErrDialWhileUnpaused is returned if Dial is called on the Manager without
// pausing it first
var ErrDialWhileUnpaused = errors.New("dial is called while manager is not paused")

// Manager manages a UDP connection. It allows reconnecting. A Manager instance
// is thread-safe, meaning it can be used concurrently.
type Manager struct {
	dialer *net.Dialer

	stopMu   sync.Mutex
	stopConn chan struct{}
	stopDial context.CancelFunc

	// conn state
	conn     *Connection
	connLock chan struct{}

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
		dialer:   d,
		stopConn: make(chan struct{}),
		connLock: make(chan struct{}, 1),
	}
}

// SetDialer sets the manager's dialer.
func (m *Manager) SetDialer(d *net.Dialer) {
	m.connLock <- struct{}{}
	m.dialer = d
	<-m.connLock
}

// Close closes the current connection. If the connection is already closed,
// then nothing is done. If the connection is being paused, then it is unpaused.
func (m *Manager) Close() (err error) {
	// Acquire the mutex first.
	m.stopMu.Lock()
	defer m.stopMu.Unlock()

	// Cancel the dialing.
	if m.stopDial != nil {
		m.stopDial()
		m.stopDial = nil
	}

	// Stop existing Manager users.
	select {
	case <-m.stopConn:
		// m.stopConn already closed
	default:
		close(m.stopConn)
	}

	// Acquire the dial lock to ensure that it's done.
	m.connLock <- struct{}{}
	<-m.connLock

	return nil
}

// Continue resumes the active users. If the manager has been successfully
// resumed, then true is returned, otherwise if it's already continued, then
// false is returned.
func (m *Manager) Continue() bool {
	select {
	case <-m.connLock:
		return true
	default:
		return false
	}
}

// PauseAndDial pauses active users of the Manager and dials the internal
// connection to the given address and SSRC number. The caller must call Dial
// after Pause and before Unpause. If the Manager is already being dialed
// elsewhere, then ErrAlreadyDialing is returned.
func (m *Manager) PauseAndDial(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
	select {
	case m.connLock <- struct{}{}:
		// acquired
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	m.stopMu.Lock()
	// Allow cancelling from another goroutine with this context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m.stopDial = cancel
	m.stopMu.Unlock()

	conn, err := DialConnectionCustom(ctx, m.dialer, addr, ssrc)
	if err != nil {
		// Unlock if we failed.
		<-m.connLock
		return nil, errors.Wrap(err, "failed to dial")
	}

	if m.frequency > 0 && m.timeIncr > 0 {
		conn.ResetFrequency(m.frequency, m.timeIncr)
	}

	m.conn = conn

	m.stopMu.Lock()
	m.stopDial = nil
	m.stopConn = make(chan struct{})
	m.stopMu.Unlock()

	return conn, nil
}

// ResetFrequency sets the current connection and future connections' write
// frequency. Note that calling this method while Connection is being used in a
// different goroutine is not thread-safe.
func (m *Manager) ResetFrequency(frameDuration time.Duration, timeIncr uint32) {
	m.connLock <- struct{}{}
	defer func() { <-m.connLock }()

	m.frequency = frameDuration
	m.timeIncr = timeIncr

	if m.conn != nil {
		m.conn.ResetFrequency(frameDuration, timeIncr)
	}
}

// ReadPacket reads the current packet. It blocks until a packet arrives or
// the Manager is closed.
func (m *Manager) ReadPacket() (p *Packet, err error) {
	conn := m.acquireConn()
	if conn == nil {
		return nil, ErrManagerClosed
	}

	return conn.ReadPacket()
}

// Write writes to the current connection in the manager. It blocks if the
// connection is being re-established.
func (m *Manager) Write(b []byte) (n int, err error) {
	conn := m.acquireConn()
	if conn == nil {
		return 0, ErrManagerClosed
	}

	return conn.Write(b)
}

// acquireConn acquires the current connection and releases the lock, returning
// the connection at that point in time. Nil is returned if Manager is closed.
func (m *Manager) acquireConn() *Connection {
	m.stopMu.Lock()
	defer m.stopMu.Unlock()

	select {
	case m.connLock <- struct{}{}:
		defer func() { <-m.connLock }()
	case <-m.stopConn:
		return nil
	}

	return m.conn
}
