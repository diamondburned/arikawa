package udp

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
)

type Connection struct {
	GatewayIP   string
	GatewayPort uint16

	ssrc uint32

	sequence  uint16
	timestamp uint32
	nonce     [24]byte

	conn   *net.UDPConn
	close  chan struct{}
	closed chan struct{}

	send  chan []byte
	reply chan error
}

func DialConnection(addr string, ssrc uint32) (*Connection, error) {
	// Resolve the host.
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve host")
	}

	// Create a new UDP connection.
	conn, err := net.DialUDP("udp", nil, a)
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

	return &Connection{
		GatewayIP:   string(ip),
		GatewayPort: port,

		ssrc:   ssrc,
		conn:   conn,
		send:   make(chan []byte),
		reply:  make(chan error),
		close:  make(chan struct{}),
		closed: make(chan struct{}),
	}, nil
}

func (c *Connection) Start(secret *[32]byte) {
	// https://discordapp.com/developers/docs/topics/voice-connections#encrypting-and-sending-voice
	packet := [12]byte{
		0: 0x80, // Version + Flags
		1: 0x78, // Payload Type
		// [2:4] // Sequence
		// [4:8] // Timestamp
	}

	// Write SSRC to the header.
	binary.BigEndian.PutUint32(packet[8:12], c.ssrc) // SSRC

	// 50 sends per second, 960 samples each at 48kHz
	frequency := time.NewTicker(time.Millisecond * 20)
	defer frequency.Stop()

	var b []byte
	var ok bool

	// Close these channels at the end so Write() doesn't block.
	defer func() {
		close(c.send)
		close(c.closed)
	}()

	for {
		select {
		case b, ok = <-c.send:
			if !ok {
				return
			}
		case <-c.close:
			return
		}

		// Write a new sequence.
		binary.BigEndian.PutUint16(packet[2:4], c.sequence)
		c.sequence++

		binary.BigEndian.PutUint32(packet[4:8], c.timestamp)
		c.timestamp += 960 // Samples

		copy(c.nonce[:], packet[:])

		toSend := secretbox.Seal(packet[:], b, &c.nonce, secret)

		select {
		case <-frequency.C:
		case <-c.close:
			return
		}

		_, err := c.conn.Write(toSend)
		c.reply <- err
	}
}

func (c *Connection) Close() error {
	close(c.close)
	<-c.closed

	return c.conn.Close()
}

// Write sends bytes into the voice UDP connection.
func (c *Connection) Write(b []byte) (int, error) {
	c.send <- b
	if err := <-c.reply; err != nil {
		return 0, err
	}
	return len(b), nil
}
