package wsutil

import (
	"fmt"
	"sync"

	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/moreatomic"
	"github.com/pkg/errors"
)

var ErrEmptyPayload = errors.New("empty payload")

// OPCode is a generic type for websocket OP codes.
type OPCode uint8

type OP struct {
	Code OPCode   `json:"op"`
	Data json.Raw `json:"d,omitempty"`

	// Only for Gateway Dispatch (op 0)
	Sequence  int64  `json:"s,omitempty"`
	EventName string `json:"t,omitempty"`
}

func (op *OP) UnmarshalData(v interface{}) error {
	return json.Unmarshal(op.Data, v)
}

func DecodeOP(ev Event) (*OP, error) {
	if ev.Error != nil {
		return nil, ev.Error
	}

	if len(ev.Data) == 0 {
		return nil, ErrEmptyPayload
	}

	var op *OP
	if err := json.Unmarshal(ev.Data, &op); err != nil {
		return nil, errors.Wrap(err, "OP error: "+string(ev.Data))
	}

	return op, nil
}

func AssertEvent(ev Event, code OPCode, v interface{}) (*OP, error) {
	op, err := DecodeOP(ev)
	if err != nil {
		return nil, err
	}

	if op.Code != code {
		return op, fmt.Errorf(
			"Unexpected OP Code: %d, expected %d (%s)",
			op.Code, code, op.Data,
		)
	}

	if err := json.Unmarshal(op.Data, v); err != nil {
		return op, errors.Wrap(err, "failed to decode data")
	}

	return op, nil
}

type EventHandler interface {
	HandleOP(op *OP) error
}

func HandleEvent(h EventHandler, ev Event) error {
	o, err := DecodeOP(ev)
	if err != nil {
		return err
	}

	return h.HandleOP(o)
}

// WaitForEvent blocks until fn() returns true. All incoming events are handled
// regardless.
func WaitForEvent(h EventHandler, ch <-chan Event, fn func(*OP) bool) error {
	for ev := range ch {
		o, err := DecodeOP(ev)
		if err != nil {
			return err
		}

		// Handle the *OP first, in case it's an Invalid Session. This should
		// also prevent a race condition with things that need Ready after
		// Open().
		if err := h.HandleOP(o); err != nil {
			return err
		}

		// Are these events what we're looking for? If we've found the event,
		// return.
		if fn(o) {
			return nil
		}
	}

	return errors.New("event not found and event channel is closed")
}

type ExtraHandlers struct {
	mutex    sync.Mutex
	handlers map[uint32]*ExtraHandler
	serial   uint32
}

type ExtraHandler struct {
	Check func(*OP) bool
	send  chan *OP

	closed moreatomic.Bool
}

func (ex *ExtraHandlers) Add(check func(*OP) bool) (<-chan *OP, func()) {
	handler := &ExtraHandler{
		Check: check,
		send:  make(chan *OP),
	}

	ex.mutex.Lock()
	defer ex.mutex.Unlock()

	if ex.handlers == nil {
		ex.handlers = make(map[uint32]*ExtraHandler, 1)
	}

	i := ex.serial
	ex.serial++

	ex.handlers[i] = handler

	return handler.send, func() {
		// Check the atomic bool before acquiring the mutex. Might help a bit in
		// performance.
		if handler.closed.Get() {
			return
		}

		ex.mutex.Lock()
		defer ex.mutex.Unlock()

		delete(ex.handlers, i)
	}
}

// Check runs and sends OP data. It is not thread-safe.
func (ex *ExtraHandlers) Check(op *OP) {
	ex.mutex.Lock()
	defer ex.mutex.Unlock()

	for i, handler := range ex.handlers {
		if handler.Check(op) {
			// Attempt to send.
			handler.send <- op

			// Mark the handler as closed.
			handler.closed.Set(true)

			// Delete the handler.
			delete(ex.handlers, i)
		}
	}
}
