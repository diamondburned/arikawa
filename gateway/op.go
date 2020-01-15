package gateway

import (
	"fmt"

	"github.com/diamondburned/arikawa/json"
	"github.com/diamondburned/arikawa/wsutil"
	"github.com/pkg/errors"
)

type OP struct {
	Code OPCode   `json:"op"`
	Data json.Raw `json:"d,omitempty"`

	// Only for Dispatch (op 0)
	Sequence  int    `json:"s,omitempty"`
	EventName string `json:"t,omitempty"`
}

func DecodeOP(driver json.Driver, ev wsutil.Event) (*OP, error) {
	if ev.Error != nil {
		return nil, ev.Error
	}

	var op *OP
	if err := driver.Unmarshal(ev.Data, &op); err != nil {
		return nil, errors.Wrap(err, "Failed to decode payload")
	}

	return op, nil
}

func DecodeEvent(driver json.Driver,
	ev wsutil.Event, v interface{}) (OPCode, error) {

	op, err := DecodeOP(driver, ev)
	if err != nil {
		return 0, err
	}

	if err := driver.Unmarshal(op.Data, v); err != nil {
		return 0, errors.Wrap(err, "Failed to decode data")
	}

	return op.Code, nil
}

func AssertEvent(driver json.Driver,
	ev wsutil.Event, code OPCode, v interface{}) error {

	op, err := DecodeOP(driver, ev)
	if err != nil {
		return err
	}

	if op.Code != code {
		return fmt.Errorf(
			"Unexpected OP Code: %d, expected %d (%s)",
			op.Code, code, op.Data,
		)
	}

	if err := driver.Unmarshal(op.Data, v); err != nil {
		return errors.Wrap(err, "Failed to decode data")
	}

	return nil
}

type OPCode uint8

const (
	DispatchOP            OPCode = iota // recv
	HeartbeatOP                         // send/recv
	IdentifyOP                          // send...
	StatusUpdateOP                      //
	VoiceStateUpdateOP                  //
	VoiceServerPingOP                   //
	ResumeOP                            //
	ReconnectOP                         // recv
	RequestGuildMembersOP               // send
	InvalidSessionOP                    // recv...
	HelloOP
	HeartbeatAckOP
	_
	CallConnectOP
	GuildSubscriptionsOP
)
