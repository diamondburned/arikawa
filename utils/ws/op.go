package ws

import (
	"context"
	"fmt"
	"sync"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

// OpCode is the type for websocket Op codes. Op codes less than 0 are
// internal Op codes and should usually be ignored.
type OpCode int

// CloseEvent is an event that is given from wsutil when the websocket is
// closed.
type CloseEvent struct {
	// Err is the underlying error.
	Err error
	// Code is the websocket close code, if any. It is -1 otherwise.
	Code int
}

// Unwrap returns err.Err.
func (e *CloseEvent) Unwrap() error { return e.Err }

// Error formats the CloseEvent. A CloseEvent is also an error.
func (e *CloseEvent) Error() string {
	return fmt.Sprintf("websocket closed, reason: %s", e.Err)
}

// Op implements Event. It returns -1.
func (e *CloseEvent) Op() OpCode { return -1 }

// EventType implements Event. It returns an emty string.
func (e *CloseEvent) EventType() EventType { return "__ws.CloseEvent" }

// EnableRawEvents, if true, will cause ws to generate a RawEvent for each
// regular Event. It should only be used for debugging.
var EnableRawEvents = false

// RawEvent is used if EnableRawEvents is true.
type RawEvent struct {
	json.Raw
	OriginalCode OpCode    `json:"-"`
	OriginalType EventType `json:"-"`
}

// Op implements Event. It returns -1.
func (e *RawEvent) Op() OpCode { return -1 }

// EventType implements Event. It returns an emty string.
func (e *RawEvent) EventType() EventType { return "__ws.RawEvent" }

// EventType is a type for event types, which is the "t" field in the payload.
type EventType string

// Event describes an Event data that comes from a gateway Operation.
type Event interface {
	Op() OpCode
	EventType() EventType
}

// OpFunc is a constructor function for an Operation.
type OpFunc func() Event

// OpUnmarshalers contains a map of event constructor function.
type OpUnmarshalers struct {
	r map[opFuncID]OpFunc
}

type opFuncID struct {
	Op OpCode    `json:"op"`
	T  EventType `json:"t"`
}

// NewOpUnmarshalers creates a nwe OpUnmarshalers instance from the given
// constructor functions.
func NewOpUnmarshalers(funcs ...OpFunc) OpUnmarshalers {
	m := OpUnmarshalers{r: make(map[opFuncID]OpFunc)}
	m.Add(funcs...)
	return m
}

// Each iterates over the marshaler map.
func (m OpUnmarshalers) Each(f func(OpCode, EventType, OpFunc) (done bool)) {
	for id, fn := range m.r {
		if f(id.Op, id.T, fn) {
			return
		}
	}
}

// Add adds the given functions into the unmarshaler registry.
func (m OpUnmarshalers) Add(funcs ...OpFunc) {
	for _, fn := range funcs {
		ev := fn()
		id := opFuncID{
			Op: ev.Op(),
			T:  ev.EventType(),
		}

		m.r[id] = fn
	}
}

// Lookup searches the OpMarshalers map for the given constructor function.
func (m OpUnmarshalers) Lookup(op OpCode, t EventType) OpFunc {
	return m.r[opFuncID{op, t}]
}

// Op is a gateway Operation.
type Op struct {
	Code OpCode `json:"op"`
	Data Event  `json:"d,omitempty"`

	// Type is only for gateway dispatch events.
	Type EventType `json:"t,omitempty"`
	// Sequence is only for gateway dispatch events (Op 0).
	Sequence int64 `json:"s,omitempty"`
}

// UnknownEventError is required by HandleOp if an event is encountered that is
// not known. Internally, unknown events are logged and ignored. It is not a
// fatal error.
type UnknownEventError struct {
	Op   OpCode
	Type EventType
}

// Error formats the unknown event error to with the event name and payload
func (err UnknownEventError) Error() string {
	return fmt.Sprintf("unknown op %d, event %s", err.Op, err.Type)
}

// IsBrokenConnection returns true if the error is a broken connection error.
func IsUnknownEvent(err error) bool {
	var uevent *UnknownEventError
	return errors.As(err, &uevent)
}

// ReadOps reads maximum n Ops and accumulate them into a slice.
func ReadOps(ctx context.Context, ch <-chan Op, n int) ([]Op, error) {
	ops := make([]Op, 0, n)
	for {
		select {
		case <-ctx.Done():
			return ops, ctx.Err()
		case op := <-ch:
			ops = append(ops, op)
			if len(ops) == n {
				return ops, nil
			}
		}
	}
}

// ReadOp reads a single Op.
func ReadOp(ctx context.Context, ch <-chan Op) (Op, error) {
	select {
	case <-ctx.Done():
		return Op{}, ctx.Err()
	case op := <-ch:
		return op, nil
	}
}

// Broadcaster is primarily used for debugging.
type Broadcaster struct {
	src  <-chan Op
	dst  map[chan<- Op]struct{}
	mut  sync.Mutex
	void bool
}

// NewBroadcaster creates a new broadcaster.
func NewBroadcaster(src <-chan Op) *Broadcaster {
	return &Broadcaster{
		src: src,
		dst: make(map[chan<- Op]struct{}),
	}
}

// Start starts the broadcasting loop.
func (b *Broadcaster) Start() {
	b.mut.Lock()
	if b.void {
		panic("Start called on voided Broadcaster")
	}
	b.mut.Unlock()

	go func() {
		for op := range b.src {
			b.mut.Lock()

			for ch := range b.dst {
				ch <- op
			}

			b.mut.Unlock()
		}

		b.mut.Lock()
		b.void = true

		for ch := range b.dst {
			close(ch)
		}

		b.mut.Unlock()
	}()
}

// Subscribe subscribes the given channel
func (b *Broadcaster) Subscribe(ch chan<- Op) {
	b.mut.Lock()
	if b.void {
		panic("Subscribe called on voided Broadcaster")
	}
	b.dst[ch] = struct{}{}
	b.mut.Unlock()
}

// NewSubscribed creates a newly subscribed Op channel.
func (b *Broadcaster) NewSubscribed() <-chan Op {
	ch := make(chan Op, 1)
	b.Subscribe(ch)
	return ch
}
