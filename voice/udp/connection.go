package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// ErrDecryptionFailed is returned from ReadPacket if the received packet fails
// to decrypt.
var ErrDecryptionFailed = errors.New("decryption failed")

// defaultDialer is the default dialer that this package uses for all its
// dialing.
var defaultDialer = net.Dialer{
	Timeout: 30 * time.Second,
}

// Connection represents a voice connection. It is not thread-safe.
type Connection struct {
	GatewayIP   string
	GatewayPort uint16

	conn net.Conn
	ssrc uint32

	// frequency rate.Limiter
	frequency *time.Ticker
	timeIncr  uint32
	stopFreq  chan struct{}

	packet [12]byte
	secret [32]byte

	sequence  uint16
	timestamp uint32
	nonce     [24]byte

	// recv fields
	recvNonce  [24]byte
	recvBuf    []byte  // len 1400
	recvOpus   []byte  // len 1400
	recvPacket *Packet // uses recvOpus' backing array

	closed sync.Once
}

// DialFunc is the UDP dialer function type. It's the function signature for
// udp.DialConnection.
type DialFunc = func(ctx context.Context, addr string, ssrc uint32) (*Connection, error)

// Assert that this is the same.
var _ DialFunc = DialConnection

// DialFuncWithFrequency creates a new DialFunc with the given frame duration
// and time increment. See Connection's ResetFrequency method for more
// information.
func DialFuncWithFrequency(frameDuration time.Duration, timeIncr uint32) DialFunc {
	return func(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
		u, err := DialConnection(ctx, addr, ssrc)
		if err != nil {
			return nil, err
		}
		u.ResetFrequency(frameDuration, timeIncr)
		return u, nil
	}
}

// DialConnection dials the UDP connection using the given address and SSRC
// number.
func DialConnection(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
	return DialConnectionCustom(ctx, &defaultDialer, addr, ssrc)
}

// DialConnectionCustom dials the UDP connection with a custom dialer.
func DialConnectionCustom(
	ctx context.Context, dialer *net.Dialer, addr string, ssrc uint32) (*Connection, error) {

	// Create a new UDP connection.
	conn, err := dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial host: %w", err)
	}

	// https://discord.com/developers/docs/topics/voice-connections#ip-discovery
	var ssrcBuffer [74]byte
	binary.BigEndian.PutUint16(ssrcBuffer[0:2], 1)
	binary.BigEndian.PutUint16(ssrcBuffer[2:4], 70)
	binary.BigEndian.PutUint32(ssrcBuffer[4:8], ssrc)

	_, err = conn.Write(ssrcBuffer[:])
	if err != nil {
		return nil, fmt.Errorf("failed to write SSRC buffer: %w", err)
	}

	var ipBuffer [74]byte

	// ReadFull makes sure to read all 74 bytes.
	_, err = io.ReadFull(conn, ipBuffer[:])
	if err != nil {
		return nil, fmt.Errorf("failed to read IP buffer: %w", err)
	}

	ipbody := ipBuffer[8:72]

	nullPos := bytes.Index(ipbody, []byte{'\x00'})
	if nullPos < 0 {
		return nil, errors.New("UDP IP discovery did not contain a null terminator")
	}

	ip := ipbody[:nullPos]
	port := binary.LittleEndian.Uint16(ipBuffer[72:74])

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
		stopFreq:    make(chan struct{}),
		packet:      packet,
		ssrc:        ssrc,
		conn:        conn,
		recvBuf:     make([]byte, 1400),
		recvOpus:    make([]byte, 1400),
		recvPacket:  &Packet{},
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
// # Valid Parameters
//
// The following table lists the recommended parameters for these variables.
//
//	+---------+-----+-----+------+------+
//	|   Mode  |  10 |  20 |  40  |  60  |
//	+---------+-----+-----+------+------+
//	| ts incr | 480 | 960 | 1920 | 2880 |
//	+---------+-----+-----+------+------+
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

// SetWriteDeadline sets the UDP connection's write deadline.
func (c *Connection) SetWriteDeadline(deadline time.Time) {
	c.conn.SetWriteDeadline(deadline)
}

// SetReadDeadline sets the UDP connection's read deadline.
func (c *Connection) SetReadDeadline(deadline time.Time) {
	c.conn.SetReadDeadline(deadline)
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.closed.Do(func() {
		// Be sure to only run this ONCE.
		c.frequency.Stop()
		close(c.stopFreq)
	})

	return c.conn.Close()
}

// Write sends a packet of audio into the voice UDP connection. It is made to be
// stream-compatible: the internal frequency clock will slow Write down to match
// the real playback time.
func (c *Connection) Write(b []byte) (int, error) {
	// Write a new sequence.
	binary.BigEndian.PutUint16(c.packet[2:4], c.sequence)
	c.sequence++

	binary.BigEndian.PutUint32(c.packet[4:8], c.timestamp)
	c.timestamp += c.timeIncr

	// Copy the first 12 bytes from the packet into the nonce.
	copy(c.nonce[:12], c.packet[:])

	// Seal the message, but reuse the packet buffer. We pass in the first 12
	// bytes of the packet, but allow it to reuse the whole packet buffer
	toSend := secretbox.Seal(c.packet[:12], b, &c.nonce, &c.secret)

	select {
	case <-c.frequency.C:
		// ok
	case <-c.stopFreq:
		return 0, fmt.Errorf("frequency ticker stopped: %w", net.ErrClosed)
	}

	_, err := c.conn.Write(toSend)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Packet represents a voice packet.
type Packet struct {
	header []byte
	Opus   []byte
}

// VersionFlags returns the version flags of the current packet.
func (p *Packet) VersionFlags() byte { return p.header[0] }

// Type returns the packet type.
func (p *Packet) Type() byte { return p.header[1] }

// Sequence returns the packet sequence.
func (p *Packet) Sequence() uint16 { return binary.BigEndian.Uint16(p.header[2:4]) }

// Timestamp returns the packet's timestamp.
func (p *Packet) Timestamp() uint32 { return binary.BigEndian.Uint32(p.header[4:8]) }

// SSRC returns the packet's SSRC number.
func (p *Packet) SSRC() uint32 { return binary.BigEndian.Uint32(p.header[8:12]) }

// Copy copies the current packet into the given packet.
func (p *Packet) Copy(dst *Packet) {
	dst.header = append(dst.header[:0], p.header...)
	dst.Opus = append(dst.Opus[:0], p.Opus...)
}

const packetHeaderSize = 12

// ReadPacket reads the UDP connection and returns a packet if successful. The
// returned packet is invalidated once ReadPacket is called again. To avoid
// this, manually Copy the packet.
func (c *Connection) ReadPacket() (*Packet, error) {
	if c.recvPacket.header == nil {
		// Initialize the recvPacket's header.
		c.recvPacket.header = c.recvBuf[:12]
	}

	for {
		i, err := c.conn.Read(c.recvBuf)
		if err != nil {
			return nil, err
		}

		if i < packetHeaderSize || (c.recvBuf[0] != 0x80 && c.recvBuf[0] != 0x90) {
			continue
		}

		// Copy the nonce to be read.
		// TODO: once Go 1.17 is released, we can remove recvNonce and directly
		// cast it as (*[packetHeaderSize]byte)(c.recvBuf).
		copy(c.recvNonce[:], c.recvBuf[0:packetHeaderSize])

		var ok bool

		// Open (decrypt) the rest of the received bytes.
		c.recvPacket.Opus, ok = secretbox.Open(
			c.recvOpus[:0], c.recvBuf[packetHeaderSize:i], &c.recvNonce, &c.secret)
		if !ok {
			return nil, ErrDecryptionFailed
		}

		// Partial structure of the RTP header for reference
		//
		//     0                   1                   2                   3
		//     0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//    |V=2|P|X|  CC   |M|     PT      |       sequence number         |
		//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//    |                           timestamp                           |
		//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		// References
		//
		//    https://tools.ietf.org/html/rfc3550#section-5.1
		//

		// We first check VersionFlags (8-bit) for whether or not the 4th bit
		// (extension) is set. The value of 0x10 is 0b00010000. RFC3550 section
		// 5.1 explains the extension bit as:
		//
		//    If the extension bit is set, the fixed header MUST be followed by
		//    exactly one header extension, with a format defined in Section
		//    5.3.1.
		//
		isExtension := c.recvPacket.VersionFlags()&0x10 == 0x10

		// We then check for whether or not the marker bit (9th bit) is set. The
		// 9th bit is carried over to the second byte (Type), so we check its
		// presence with 0x80, or 0b10000000. RFC3550 section 5.1 explains the
		// marker bit as:
		//
		//     The interpretation of the marker is defined by a profile.  It is
		//     intended to allow significant events such as frame boundaries to
		//     be marked in the packet stream.  A profile MAY define additional
		//     marker bits or specify that there is no marker bit by changing
		//     the number of bits in the payload type field (see Section 5.3).
		//
		// RFC3350 section 12.1 also writes:
		//
		//    When the RTCP packet type field is compared to the corresponding
		//    octet of the RTP header, this range corresponds to the marker bit
		//    being 1 (which it usually is not in data packets) and to the high
		//    bit of the standard payload type field being 1 (since the static
		//    payload types are typically defined in the low half).
		//
		// This implies that, when the marker bit is 1, the received packet is
		// an RTCP packet and NOT an RTP packet; therefore, we must ignore the
		// unknown sections, so we do a (NOT isMarker) check below.
		isMarker := c.recvPacket.Type()&0x80 != 0x0

		if isExtension && !isMarker {
			extLen := binary.BigEndian.Uint16(c.recvPacket.Opus[2:4])
			shift := 4 + 4*int(extLen)

			if len(c.recvPacket.Opus) > shift {
				c.recvPacket.Opus = c.recvPacket.Opus[shift:]
			}
		}

		return c.recvPacket, nil
	}
}
