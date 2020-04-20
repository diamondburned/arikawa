package voice

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
)

// udpOpen .
func (c *Connection) udpOpen() error {
	c.mut.Lock()
	defer c.mut.Unlock()

	// As a wise man once said: "You always gotta check for stupidity"
	if c.WS == nil {
		return errors.New("connection does not have a websocket")
	}

	// Check if a UDP connection is already open.
	if c.udpConn != nil {
		return errors.New("udp connection is already open")
	}

	// Format the connection host.
	host := c.ready.IP + ":" + strconv.Itoa(c.ready.Port)

	// Resolve the host.
	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return errors.Wrap(err, "Failed to resolve host")
	}

	// Create a new UDP connection.
	c.udpConn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		return errors.Wrap(err, "Failed to dial host")
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#ip-discovery
	ssrcBuffer := make([]byte, 70)
	ssrcBuffer[0] = 0x1
	ssrcBuffer[1] = 0x2
	binary.BigEndian.PutUint16(ssrcBuffer[2:4], 70)
	binary.BigEndian.PutUint32(ssrcBuffer[4:8], c.ready.SSRC)
	_, err = c.udpConn.Write(ssrcBuffer)
	if err != nil {
		return errors.Wrap(err, "Failed to write")
	}

	ipBuffer := make([]byte, 70)
	var n int
	n, err = c.udpConn.Read(ipBuffer)
	if err != nil {
		return errors.Wrap(err, "Failed to write")
	}
	if n < 70 {
		return errors.New("udp packet received from discord is not the required 70 bytes")
	}

	ipb := string(ipBuffer[4:68])
	nullPos := strings.Index(ipb, "\x00")
	if nullPos < 0 {
		return errors.New("udp ip discovery did not contain a null terminator")
	}
	ip := ipb[:nullPos]
	port := binary.LittleEndian.Uint16(ipBuffer[68:70])

	// Send a Select Protocol operation to the Discord Voice Gateway.
	err = c.SelectProtocol(SelectProtocol{
		Protocol: "udp",
		Data: SelectProtocolData{
			Address: ip,
			Port:    port,
			Mode:    "xsalsa20_poly1305",
		},
	})
	if err != nil {
		return err
	}

	// TODO: Wait until OP4 is received
	// side note: you cannot just do a blocking loop as I've done before
	// as this method is currently called inside of the event loop
	// so for as long as it blocks no other events can be received

	return nil
}

// https://discordapp.com/developers/docs/topics/voice-connections#encrypting-and-sending-voice
func (c *Connection) opusSendLoop() {
	header := make([]byte, 12)
	header[0] = 0x80 // Version + Flags
	header[1] = 0x78 // Payload Type
	// header[2:4] // Sequence
	// header[4:8] // Timestamp
	binary.BigEndian.PutUint32(header[8:12], c.ready.SSRC) // SSRC

	var (
		sequence  uint16
		timestamp uint32
		nonce     [24]byte

		msg  []byte
		open bool
	)

	// 50 sends per second, 960 samples each at 48kHz
	frequency := time.NewTicker(time.Millisecond * 20)
	defer frequency.Stop()

	for {
		select {
		case msg, open = <-c.OpusSend:
			if !open {
				return
			}
		case <-c.close:
			return
		}

		binary.BigEndian.PutUint16(header[2:4], sequence)
		sequence++

		binary.BigEndian.PutUint32(header[4:8], timestamp)
		timestamp += 960 // Samples

		copy(nonce[:], header)

		toSend := secretbox.Seal(header, msg, &nonce, &c.sessionDescription.SecretKey)
		select {
		case <-frequency.C:
		case <-c.close:
			return
		}

		_, _ = c.udpConn.Write(toSend)
	}
}
