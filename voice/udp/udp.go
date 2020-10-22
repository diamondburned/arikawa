package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/time/rate"
)

// Dialer is the default dialer that this package uses for all its dialing.
var Dialer = net.Dialer{
	Timeout: 10 * time.Second,
}

// ErrClosed is returned if a Write was called on a closed connection.
var ErrClosed = errors.New("UDP connection closed")

type Connection struct {
	GatewayIP   string
	GatewayPort uint16

	mutex chan struct{} // for ctx

	context context.Context
	conn    net.Conn
	ssrc    uint32

	frequency rate.Limiter
	packet    [12]byte
	secret    [32]byte

	sequence  uint16
	timestamp uint32
	nonce     [24]byte
}

func DialConnectionCtx(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
	// Create a new UDP connection.
	conn, err := Dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial host")
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#ip-discovery
	ssrcBuffer := [70]byte{
		0x1, 0x2,
	}
	binary.BigEndian.PutUint16(ssrcBuffer[2:4], 70)
	binary.BigEndian.PutUint32(ssrcBuffer[4:8], ssrc)

	_, err = conn.Write(ssrcBuffer[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to write SSRC buffer")
	}

	var ipBuffer [70]byte

	// ReadFull makes sure to read all 70 bytes.
	_, err = io.ReadFull(conn, ipBuffer[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to read IP buffer")
	}

	ipbody := ipBuffer[4:68]

	nullPos := bytes.Index(ipbody, []byte{'\x00'})
	if nullPos < 0 {
		return nil, errors.New("UDP IP discovery did not contain a null terminator")
	}

	ip := ipbody[:nullPos]
	port := binary.LittleEndian.Uint16(ipBuffer[68:70])

	// https://discordapp.com/developers/docs/topics/voice-connections#encrypting-and-sending-voice
	packet := [12]byte{
		0: 0x80, // Version + Flags
		1: 0x78, // Payload Type
		// [2:4] // Sequence
		// [4:8] // Timestamp
	}

	// Write SSRC to the header.
	binary.BigEndian.PutUint32(packet[8:12], ssrc) // SSRC

	return &Connection{
		GatewayIP:   string(ip),
		GatewayPort: port,
		// 50 sends per second, 960 samples each at 48kHz
		frequency: *rate.NewLimiter(rate.Every(20*time.Millisecond), 1),
		context:   context.Background(),
		mutex:     make(chan struct{}, 1),
		packet:    packet,
		ssrc:      ssrc,
		conn:      conn,
	}, nil
}

// UseSecret uses the given secret. This method is not thread-safe, so it should
// only be used right after initialization.
func (c *Connection) UseSecret(secret [32]byte) {
	c.secret = secret
}

// UseContext lets the connection use the given context for its Write method.
// WriteCtx will override this context.
func (c *Connection) UseContext(ctx context.Context) error {
	c.mutex <- struct{}{}
	defer func() { <-c.mutex }()

	return c.useContext(ctx)
}

func (c *Connection) useContext(ctx context.Context) error {
	if c.conn == nil {
		return ErrClosed
	}

	if c.context == ctx {
		return nil
	}

	c.context = ctx

	if deadline, ok := c.context.Deadline(); ok {
		return c.conn.SetWriteDeadline(deadline)
	} else {
		return c.conn.SetWriteDeadline(time.Time{})
	}
}

func (c *Connection) Close() error {
	c.mutex <- struct{}{}
	err := c.conn.Close()
	c.conn = nil
	<-c.mutex
	return err
}

// Write sends bytes into the voice UDP connection.
func (c *Connection) Write(b []byte) (int, error) {
	select {
	case c.mutex <- struct{}{}:
		defer func() { <-c.mutex }()
	case <-c.context.Done():
		return 0, c.context.Err()
	}

	if c.conn == nil {
		return 0, ErrClosed
	}

	return c.write(b)
}

// WriteCtx sends bytes into the voice UDP connection with a timeout.
func (c *Connection) WriteCtx(ctx context.Context, b []byte) (int, error) {
	select {
	case c.mutex <- struct{}{}:
		defer func() { <-c.mutex }()
	case <-c.context.Done():
		return 0, c.context.Err()
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	if err := c.useContext(ctx); err != nil {
		return 0, errors.Wrap(err, "failed to use context")
	}

	return c.write(b)
}

// write is thread-unsafe.
func (c *Connection) write(b []byte) (int, error) {
	// Write a new sequence.
	binary.BigEndian.PutUint16(c.packet[2:4], c.sequence)
	c.sequence++

	binary.BigEndian.PutUint32(c.packet[4:8], c.timestamp)
	c.timestamp += 960 // Samples

	copy(c.nonce[:], c.packet[:])

	if err := c.frequency.Wait(c.context); err != nil {
		return 0, errors.Wrap(err, "failed to wait for frequency tick")
	}

	toSend := secretbox.Seal(c.packet[:], b, &c.nonce, &c.secret)

	n, err := c.conn.Write(toSend)
	if err != nil {
		return n, errors.Wrap(err, "failed to write to UDP connection")
	}

	// We're not really returning everything, since we're "sealing" the bytes.
	return len(b), nil
}
