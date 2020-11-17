package voicegateway

import (
	"fmt"
	"sync"

	"github.com/diamondburned/arikawa/v2/utils/json"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/pkg/errors"
)

// OPCode represents a Discord Gateway Gateway operation code.
type OPCode = wsutil.OPCode

const (
	IdentifyOP           OPCode = 0  // send
	SelectProtocolOP     OPCode = 1  // send
	ReadyOP              OPCode = 2  // receive
	HeartbeatOP          OPCode = 3  // send
	SessionDescriptionOP OPCode = 4  // receive
	SpeakingOP           OPCode = 5  // send/receive
	HeartbeatAckOP       OPCode = 6  // receive
	ResumeOP             OPCode = 7  // send
	HelloOP              OPCode = 8  // receive
	ResumedOP            OPCode = 9  // receive
	ClientConnectOP      OPCode = 12 // receive
	ClientDisconnectOP   OPCode = 13 // receive
)

func (c *Gateway) HandleOP(op *wsutil.OP) error {
	wsutil.WSDebug("Handle OP", op.Code)
	switch op.Code {
	// Gives information required to make a UDP connection
	case ReadyOP:
		if err := unmarshalMutex(op.Data, &c.ready, &c.mutex); err != nil {
			return errors.Wrap(err, "failed to parse READY event")
		}

		c.Events <- &c.ready

	// Gives information about the encryption mode and secret key for sending voice packets
	case SessionDescriptionOP:
		// ?
		// Already handled by Session.

	// Someone started or stopped speaking.
	case SpeakingOP:
		ev := new(SpeakingEvent)

		if err := json.Unmarshal(op.Data, ev); err != nil {
			return errors.Wrap(err, "failed to parse Speaking event")
		}

		c.Events <- ev

	// Heartbeat response from the server
	case HeartbeatAckOP:
		c.EventLoop.Echo()

	// Hello server, we hear you! :)
	case HelloOP:
		// ?
		// Already handled on initial connection.

	// Server is saying the connection was resumed, no data here.
	case ResumedOP:
		wsutil.WSDebug("Gateway connection has been resumed.")

	case ClientConnectOP:
		ev := new(ClientConnectEvent)

		if err := json.Unmarshal(op.Data, ev); err != nil {
			return errors.Wrap(err, "failed to parse Speaking event")
		}

		c.Events <- ev

	case ClientDisconnectOP:
		ev := new(ClientDisconnectEvent)

		if err := json.Unmarshal(op.Data, ev); err != nil {
			return errors.Wrap(err, "failed to parse Speaking event")
		}

		c.Events <- ev

	default:
		return fmt.Errorf("unknown OP code %d", op.Code)
	}

	return nil
}

func unmarshalMutex(d []byte, v interface{}, m *sync.RWMutex) error {
	m.Lock()
	err := json.Unmarshal(d, v)
	m.Unlock()
	return err
}
