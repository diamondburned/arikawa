// Package handler handles incoming Gateway events. It reflects the function's
// first argument and caches that for use in each event.
//
// # Performance
//
// Each call to the event would take 167 ns/op for roughly each handler. Scaling
// that up to 100 handlers is roughly the same as multiplying 167 ns by 100,
// which gives 16700 ns or 0.0167 ms.
//
//	BenchmarkReflect-8  7260909  167 ns/op
//
// # Usage
//
// Handler's usage is mostly similar to Discordgo, in that AddHandler expects a
// function with only one argument or an event channel. For more information,
// refer to AddHandler.
package handler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Handler is a container for command handlers. A zero-value instance is a valid
// instance.
type Handler struct {
	mutex  sync.RWMutex
	events map[reflect.Type]slab // nil type for interfaces
}

func New() *Handler {
	return &Handler{}
}

// Call calls all handlers with the given event. This is an internal method; use
// with care.
func (h *Handler) Call(ev interface{}) {
	t := reflect.TypeOf(ev)

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	typedHandlers := h.events[t].Entries
	anyHandlers := h.events[nil].Entries

	if len(typedHandlers) == 0 && len(anyHandlers) == 0 {
		return
	}

	v := reflect.ValueOf(ev)

	for _, entry := range typedHandlers {
		if entry.isInvalid() {
			continue
		}
		entry.Call(v)
	}

	for _, entry := range anyHandlers {
		if entry.isInvalid() || entry.not(t) {
			continue
		}
		entry.Call(v)
	}
}

// WaitFor blocks until there's an event. It's advised to use ChanFor instead,
// as WaitFor may skip some events if it's not ran fast enough after the event
// arrived.
func (h *Handler) WaitFor(ctx context.Context, fn func(interface{}) bool) interface{} {
	var result = make(chan interface{})

	cancel := h.AddHandler(func(v interface{}) {
		if fn(v) {
			result <- v
		}
	})
	defer cancel()

	select {
	case r := <-result:
		return r
	case <-ctx.Done():
		return nil
	}
}

// ChanFor returns a channel that would receive all incoming events that match
// the callback given. The cancel() function removes the handler and drops all
// hanging goroutines.
//
// This method is more intended to be used as a filter. For a persistent event
// channel, consider adding it directly as a handler with AddHandler.
func (h *Handler) ChanFor(fn func(interface{}) bool) (out <-chan interface{}, cancel func()) {
	result := make(chan interface{})
	closer := make(chan struct{})

	removeHandler := h.AddHandler(func(v interface{}) {
		if fn(v) {
			select {
			case result <- v:
			case <-closer:
			}
		}
	})

	// Only allow cancel to be called once.
	var once sync.Once
	cancel = func() {
		once.Do(func() {
			removeHandler()
			close(closer)
		})
	}
	out = result

	return
}

// AddHandler adds the handler, returning a function that would remove this
// handler when called. A handler type is either a single-argument no-return
// function or a channel.
//
// # Function
//
// A handler can be a function with a single argument that is the expected event
// type. It must not have any returns or any other number of arguments.
//
//	// An example of a valid function handler.
//	h.AddHandler(func(*gateway.MessageCreateEvent) {})
//
// # Channel
//
// A handler can also be a channel. The underlying type that the channel wraps
// around will be the event type. As such, the type rules are the same as
// function handlers.
//
// Keep in mind that the user must NOT close the channel. In fact, the channel
// should not be closed at all. The caller function WILL PANIC if the channel is
// closed!
//
// When the rm callback that is returned is called, it will also guarantee that
// all blocking sends will be cancelled. This helps prevent dangling goroutines.
//
//	// An example of a valid channel handler.
//	ch := make(chan *gateway.MessageCreateEvent)
//	h.AddHandler(ch)
func (h *Handler) AddHandler(handler interface{}) (rm func()) {
	rm, err := h.addHandler(handler, false)
	if err != nil {
		panic(err)
	}
	return rm
}

// AddSyncHandler is a synchronous variant of AddHandler. Handlers added using
// this method will block the Call method, which is helpful if the user needs to
// rely on the order of events arriving. Handlers added using this method should
// not block for very long, as it may clog up other handlers.
func (h *Handler) AddSyncHandler(handler interface{}) (rm func()) {
	rm, err := h.addHandler(handler, true)
	if err != nil {
		panic(err)
	}
	return rm
}

// AddHandlerCheck adds the handler, but safe-guards reflect panics with a
// recoverer, returning the error. Refer to AddHandler for more information.
func (h *Handler) AddHandlerCheck(handler interface{}) (rm func(), err error) {
	// Reflect would actually panic if anything goes wrong, so this is just in
	// case.
	defer func() {
		if rec := recover(); rec != nil {
			if recErr, ok := rec.(error); ok {
				err = recErr
			} else {
				err = fmt.Errorf("%v", rec)
			}
		}
	}()

	return h.addHandler(handler, false)
}

// AddSyncHandlerCheck is the safe-guarded version of AddSyncHandler. It is
// similar to AddHandlerCheck.
func (h *Handler) AddSyncHandlerCheck(handler interface{}) (rm func(), err error) {
	// Reflect would actually panic if anything goes wrong, so this is just in
	// case.
	defer func() {
		if rec := recover(); rec != nil {
			if recErr, ok := rec.(error); ok {
				err = recErr
			} else {
				err = fmt.Errorf("%v", rec)
			}
		}
	}()

	return h.addHandler(handler, true)
}

func (h *Handler) addHandler(fn interface{}, sync bool) (rm func(), err error) {
	// Reflect the handler
	r, err := newHandler(fn, sync)
	if err != nil {
		return nil, fmt.Errorf("handler reflect failed: %w", err)
	}

	var id int
	var t reflect.Type
	if !r.isIface {
		t = r.event
	}

	h.mutex.Lock()

	if h.events == nil {
		h.events = make(map[reflect.Type]slab, 10)
	}

	slab := h.events[t]
	id = slab.Put(r)
	h.events[t] = slab

	h.mutex.Unlock()

	return func() {
		h.mutex.Lock()
		slab := h.events[t]
		popped := slab.Pop(id)
		h.mutex.Unlock()

		popped.cleanup()
	}, nil
}

type handler struct {
	event     reflect.Type // underlying type; arg0 or chan underlying type
	callback  reflect.Value
	chanclose reflect.Value // IsValid() if chan
	isIface   bool
	isSync    bool
	isOnce    bool
}

// newHandler reflects either a channel or a function into a handler. A function
// must only have a single argument being the event and no return, and a channel
// must have the event type as the underlying type.
func newHandler(unknown interface{}, sync bool) (handler, error) {
	fnV := reflect.ValueOf(unknown)
	fnT := fnV.Type()

	// underlying event type
	handler := handler{
		callback: fnV,
		isSync:   sync,
	}

	switch fnT.Kind() {
	case reflect.Func:
		if fnT.NumIn() != 1 {
			return handler, errors.New("function can only accept 1 event as argument")
		}

		if fnT.NumOut() > 0 {
			return handler, errors.New("function can't accept returns")
		}

		handler.event = fnT.In(0)

	case reflect.Chan:
		handler.event = fnT.Elem()
		handler.chanclose = reflect.ValueOf(make(chan struct{}))

	default:
		return handler, errors.New("given interface is not a function or channel")
	}

	var kind = handler.event.Kind()

	// Accept either pointer type or interface{} type
	if kind != reflect.Ptr && kind != reflect.Interface {
		return handler, errors.New("first argument is not pointer")
	}

	handler.isIface = kind == reflect.Interface

	return handler, nil
}

func (h handler) not(event reflect.Type) bool {
	if h.isIface {
		return !event.Implements(h.event)
	}

	return h.event != event
}

func (h handler) Call(event reflect.Value) {
	if h.isSync {
		h.call(event)
	} else {
		go h.call(event)
	}
}

func (h handler) call(event reflect.Value) {
	if h.chanclose.IsValid() {
		reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectSend, Chan: h.callback, Send: event},
			{Dir: reflect.SelectRecv, Chan: h.chanclose},
		})
	} else {
		h.callback.Call([]reflect.Value{event})
	}
}

func (h handler) cleanup() {
	if h.chanclose.IsValid() {
		// Closing this channel will force all ongoing selects to return
		// immediately.
		h.chanclose.Close()
	}
}
