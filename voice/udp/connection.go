package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
)

// ErrDecryptionFailed is returned from ReadPacket if the received packet fails
// to decrypt.
var ErrDecryptionFailed = errors.New("decryption failed")

// Dialer is the default dialer that this package uses for all its dialing.
var Dialer = net.Dialer{
	Timeout: 10 * time.Second,
}

// Connection represents a voice connection. It is not thread-safe.
type Connection struct {
	GatewayIP   string
	GatewayPort uint16

	conn net.Conn
	ssrc uint32

	secret [32]byte

	freqStop  chan struct{}
	frequency *time.Ticker
	timeIncr  uint32

	// recv fields
	recvNonce  [24]byte
	recvBuffer []byte // len 1400
	recvOpus   []byte // len 1400
	recvPacket Packet // uses recvOpus' and recvBuffer's backing array

	// send fields
	sendPacket []byte

	sequence  uint16
	timestamp uint32
	nonce     [24]byte

	closed chan struct{}
}

// DialConnection dials the UDP connection using the given address and SSRC
// number.
func DialConnection(ctx context.Context, addr string, ssrc uint32) (*Connection, error) {
	return DialConnectionCustom(ctx, &Dialer, addr, ssrc)
}

// DialConnectionCustom dials the UDP connection with a custom dialer.
func DialConnectionCustom(
	ctx context.Context, dialer *net.Dialer, addr string, ssrc uint32) (*Connection, error) {

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
	packet := make([]byte, 12)
	packet[0] = 0x80 // Version + Flags
	packet[1] = 0x78 // Payload Type
	// packet[2:4]  - Sequence
	// packet[4:8]  - Timestamp
	// packet[8:12] - SSRC
	// packet[12:] contains the encrypted buffer.

	// Write SSRC to the header.
	binary.BigEndian.PutUint32(packet[8:12], ssrc) // SSRC

	return &Connection{
		GatewayIP:   string(ip),
		GatewayPort: port,
		freqStop:    make(chan struct{}),
		frequency:   time.NewTicker(20 * time.Millisecond),
		timeIncr:    960,
		ssrc:        ssrc,
		conn:        conn,
		recvBuffer:  make([]byte, 1400),
		recvOpus:    make([]byte, 1400),
		sendPacket:  packet,
		closed:      make(chan struct{}),
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

// Close closes the connection.
func (c *Connection) Close() error {
	if c.IsClosed() {
		return nil
	}

	wsutil.WSDebug("UDP connection closed")
	close(c.closed)

	c.frequency.Stop()
	close(c.freqStop)
	return c.conn.Close()
}

// Write sends bytes into the voice UDP connection. Write is made to be
// stream-compatible: the internal frequency clock will slow Write down to match
// the real playback time.
func (c *Connection) Write(b []byte) (int, error) {
	// Write a new sequence.
	binary.BigEndian.PutUint16(c.sendPacket[2:4], c.sequence)
	c.sequence++

	binary.BigEndian.PutUint32(c.sendPacket[4:8], c.timestamp)
	c.timestamp += c.timeIncr

	// Copy the first 12 bytes from the packet into the nonce.
	copy(c.nonce[:12], c.sendPacket)

	// Seal the message, but reuse the packet buffer. We pass in the first 12
	// bytes of the packet, but allow it to reuse the whole packet buffer
	toSend := secretbox.Seal(c.sendPacket[:12], b, &c.nonce, &c.secret)
	// Reuse the first 12 bytes of the potentially new backing array.
	c.sendPacket = toSend[:12]

	select {
	case <-c.frequency.C:
		// continue
	case <-c.freqStop:
		return 0, errors.Wrap(net.ErrClosed, "frequency ticker stopped")
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

// IsClosed returns whether the connection is closed.
func (c *Connection) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

const packetHeaderSize = 12

// ReadPacket reads the UDP connection and returns a packet if successful. The
// returned packet is invalidated once ReadPacket is called again. To avoid
// this, manually Copy the packet.
func (c *Connection) ReadPacket() (*Packet, error) {
	if c.recvPacket.header == nil {
		// Initialize the recvPacket's header.
		c.recvPacket.header = c.recvBuffer[:12]
	}

	for {
		wsutil.WSDebug("reading from UDP connection:", c.conn.LocalAddr(), "->", c.conn.RemoteAddr())
		i, err := c.conn.Read(c.recvBuffer)
		if err != nil {
			wsutil.WSDebug("error reading from UDP connection:", err)
			return nil, err
		}

		if i < packetHeaderSize || (c.recvBuffer[0] != 0x80 && c.recvBuffer[0] != 0x90) {
			continue
		}

		// Copy the nonce to be read.
		// TODO: once Go 1.17 is released, we can remove recvNonce and directly
		// cast it as (*[packetHeaderSize]byte)(c.recvBuffer).
		copy(c.recvNonce[:], c.recvBuffer[0:packetHeaderSize])

		var ok bool

		// Open (decrypt) the rest of the received bytes.
		c.recvPacket.Opus, ok = secretbox.Open(
			c.recvOpus[:0], c.recvBuffer[packetHeaderSize:i], &c.recvNonce, &c.secret)
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

		return &c.recvPacket, nil
	}
}
