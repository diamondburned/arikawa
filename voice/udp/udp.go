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
)

// Dialer is the default dialer that this package uses for all its dialing.
var Dialer = net.Dialer{
	Timeout: 10 * time.Second,
}

// Connection represents a voice connection. It is not thread-safe.
type Connection struct {
	GatewayIP   string
	GatewayPort uint16

	context context.Context
	conn    net.Conn
	ssrc    uint32

	// frequency rate.Limiter
	frequency *time.Ticker
	timeIncr  uint32

	packet [12]byte
	secret [32]byte

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

	// https://discord.com/developers/docs/topics/voice-connections#ip-discovery
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

	// https://discord.com/developers/docs/topics/voice-connections#encrypting-and-sending-voice
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
		frequency:   time.NewTicker(20 * time.Millisecond),
		timeIncr:    960,
		context:     context.Background(),
		packet:      packet,
		ssrc:        ssrc,
		conn:        conn,
	}, nil
}

// ResetFrequency resets the internal frequency ticker as well as the timestamp
// incremental number. For more information, refer to
// https://tools.ietf.org/html/rfc7587#section-4.2.
//
// frameDuration controls the Opus frame duration used by the UDP connection to
// control the frequency of packets sent over. 20ms is the default by libopus.
//
// timestampIncr is the timestamp to increment for each Opus packet. This should
// be consistent with th given frameDuration. For the right combination, refer
// to the Valid Parameters section below.
//
// Valid Parameters
//
// The following table lists the recommended parameters for these variables.
//
//    +---------+-----+-----+------+------+
//    |   Mode  |  10 |  20 |  40  |  60  |
//    +---------+-----+-----+------+------+
//    | ts incr | 480 | 960 | 1920 | 2880 |
//    +---------+-----+-----+------+------+
//
// Note that audio mode is omitted, as it is not recommended. For the full
// table, refer to the IETF RFC7587 section 4.2 link above.
func (c *Connection) ResetFrequency(frameDuration time.Duration, timeIncr uint32) {
	c.frequency.Stop()
	c.frequency = time.NewTicker(frameDuration)
	c.timeIncr = timeIncr
}

// UseSecret uses the given secret. This method is not thread-safe, so it should
// only be used right after initialization.
func (c *Connection) UseSecret(secret [32]byte) {
	c.secret = secret
}

// UseContext lets the connection use the given context for its Write method.
// WriteCtx will override this context.
func (c *Connection) UseContext(ctx context.Context) error {
	return c.useContext(ctx)
}

func (c *Connection) useContext(ctx context.Context) error {
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
	c.frequency.Stop()
	return c.conn.Close()
}

// Write sends bytes into the voice UDP connection using the preset context.
func (c *Connection) Write(b []byte) (int, error) {
	return c.write(b)
}

// WriteCtx sends bytes into the voice UDP connection with a timeout using the
// given context. It ignores the context inside the connection, but will restore
// the deadline after this call is done.
func (c *Connection) WriteCtx(ctx context.Context, b []byte) (int, error) {
	oldCtx := c.context

	c.useContext(ctx)
	defer c.useContext(oldCtx)

	return c.write(b)
}

func (c *Connection) write(b []byte) (int, error) {
	// Write a new sequence.
	binary.BigEndian.PutUint16(c.packet[2:4], c.sequence)
	c.sequence++

	binary.BigEndian.PutUint32(c.packet[4:8], c.timestamp)
	c.timestamp += c.timeIncr

	copy(c.nonce[:], c.packet[:])

	toSend := secretbox.Seal(c.packet[:], b, &c.nonce, &c.secret)

	select {
	case <-c.frequency.C:

	case <-c.context.Done():
		return 0, c.context.Err()
	}

	n, err := c.conn.Write(toSend)
	if err != nil {
		return n, errors.Wrap(err, "failed to write to UDP connection")
	}

	// We're not really returning everything, since we're "sealing" the bytes.
	return len(b), nil
}
