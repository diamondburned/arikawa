package udp

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/pkg/errors"
)

// ErrManagerClosed is returned when a Manager that is already closed is dialed,
// written to or read from.
var ErrManagerClosed = errors.New("UDP connection manager is closed")

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

// NewManager creates a new UDP connection manager with the defalt dialer.
func NewManager() *Manager {
	return &Manager{
		dialer:   &Dialer,
		stopConn: make(chan struct{}),
		connLock: make(chan struct{}, 1),
	}
}

// SetDialer sets the manager's dialer. Calling this function while the Manager
// is working will cause a panic. Only call this method directly after
// construction.
func (m *Manager) SetDialer(d *net.Dialer) {
	select {
	case m.connLock <- struct{}{}:
		m.dialer = d
		<-m.connLock
	default:
		panic("SetDialer called while Manager is working")
	}
}

// Pause explicitly pauses the manager. It blocks until the Manager is paused or
// the context expires.
func (m *Manager) Pause(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.connLock <- struct{}{}:
		return nil
	}
}

// Close closes the current connection. If the connection is already closed,
// then nothing is done and ErrManagerClosed is returned. Close does not pause
// the connection; calls to Close while the user is using the connection will
// result in the user getting ErrManagerClosed.
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
		ws.WSDebug("UDP manager already closed")
		return ErrManagerClosed
	default:
		close(m.stopConn)
		ws.WSDebug("UDP manager closed")
	}

	return nil
}

// IsClosed returns true if the connection is closed.
func (m *Manager) IsClosed() bool {
	return m.acquireConn() == nil
}

// Continue unpauses and resumes the active user. If the manager has been
// successfully resumed, then true is returned, otherwise if it's already
// continued, then false is returned.
func (m *Manager) Continue() bool {
	ws.WSDebug("UDP continued")

	select {
	case <-m.connLock:
		return true
	default:
		return false
	}
}

// Dial dials the internal connection to the given address and SSRC number. If
// the Manager is not Paused, then an error is returned. The caller must call
// Dial after Pause and before Unpause. If the Manager is already being dialed
// elsewhere, then ErrAlreadyDialing is returned.
func (m *Manager) Dial(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
	select {
	case m.connLock <- struct{}{}:
		return nil, errors.New("Dial called on unpaused Manager")
	default:
		// ok
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

	m.stopMu.Lock()
	ws.WSDebug("setting UDP conn to one w/ gateway address", conn.GatewayIP)
	m.conn = conn
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
	// Acquire the pause lock first. We must only rely on the stopConn being
	// closed once we have this.
	m.connLock <- struct{}{}
	defer func() { <-m.connLock }()

	m.stopMu.Lock()
	defer m.stopMu.Unlock()

	select {
	case <-m.stopConn:
		ws.WSDebug("UDP acquisition got stopped conn")
		return nil
	default:
		// ok
	}

	if m.conn == nil {
		ws.WSDebug("UDP acquisition got nil conn")
	}

	return m.conn
}
