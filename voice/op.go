package voice

import (
	"fmt"

	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/pkg/errors"
)

// OPCode represents a Discord Voice Gateway operation code.
type OPCode uint8

const (
	IdentifyOP           OPCode = 0 // send
	SelectProtocolOP     OPCode = 1 // send
	ReadyOP              OPCode = 2 // receive
	HeartbeatOP          OPCode = 3 // send
	SessionDescriptionOP OPCode = 4 // receive
	SpeakingOP           OPCode = 5 // send/receive
	HeartbeatAckOP       OPCode = 6 // receive
	ResumeOP             OPCode = 7 // send
	HelloOP              OPCode = 8 // receive
	ResumedOP            OPCode = 9 // receive
	// ClientDisconnectOP   OPCode = 13 // receive
)

// OP represents a Discord Voice Gateway operation.
type OP struct {
	Code OPCode   `json:"op"`
	Data json.Raw `json:"d,omitempty"`
}

func HandleEvent(c *Connection, ev wsutil.Event) error {
	o, err := DecodeOP(c.Driver, ev)
	if err != nil {
		return err
	}

	return HandleOP(c, o)
}

func AssertEvent(driver json.Driver, ev wsutil.Event, code OPCode, v interface{}) (*OP, error) {
	op, err := DecodeOP(driver, ev)
	if err != nil {
		return nil, err
	}

	if op.Code != code {
		return op, fmt.Errorf(
			"Unexpected OP Code: %d, expected %d (%s)",
			op.Code, code, op.Data,
		)
	}

	if err := driver.Unmarshal(op.Data, v); err != nil {
		return op, errors.Wrap(err, "Failed to decode data")
	}

	return op, nil
}

func DecodeOP(driver json.Driver, ev wsutil.Event) (*OP, error) {
	if ev.Error != nil {
		return nil, ev.Error
	}

	if len(ev.Data) == 0 {
		return nil, errors.New("Empty payload")
	}

	var op *OP
	if err := driver.Unmarshal(ev.Data, &op); err != nil {
		return nil, errors.Wrap(err, "OP error: "+string(ev.Data))
	}

	return op, nil
}

func HandleOP(c *Connection, op *OP) error {
	switch op.Code {
	// Gives information required to make a UDP connection
	case ReadyOP:
		if err := c.Driver.Unmarshal(op.Data, &c.ready); err != nil {
			return errors.Wrap(err, "Failed to parse READY event")
		}

		c.readyChan <- true

	// Gives information about the encryption mode and secret key for sending voice packets
	case SessionDescriptionOP:
		if err := c.Driver.Unmarshal(op.Data, &c.sessionDescription); err != nil {
			c.ErrorLog(errors.Wrap(err, "Failed to parse SESSION_DESCRIPTION"))
		}

		c.sessionDescChan <- true

	// Someone started or stopped speaking.
	case SpeakingOP:
		// ?

	// Heartbeat response from the server
	case HeartbeatAckOP:
		c.EventLoop.Echo()

	// Hello server, we hear you! :)
	case HelloOP:
		// Decode the data.
		if err := c.Driver.Unmarshal(op.Data, &c.hello); err != nil {
			c.ErrorLog(errors.Wrap(err, "Failed to parse HELLO"))
		}

		c.helloChan <- true

	// Server is saying the connection was resumed, no data here.
	case ResumedOP:
		WSDebug("Voice connection has been resumed")

	default:
		return fmt.Errorf("unknown OP code %d", op.Code)
	}

	return nil
}
