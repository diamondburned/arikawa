// Package handler handles incoming Gateway events. It reflects the function's
// first argument and caches that for use in each event.
//
// Performance
//
// Each call to the event would take 156 ns/op for roughly each handler. Scaling
// that up to 100 handlers is multiplying 156 ns by 100, which gives 15600 ns,
// or 0.0156 ms.
//
//    BenchmarkReflect-8  7260909  156 ns/op
//
// Usage
//
// Handler's usage is similar to discordgo, in that AddHandler expects a
// function with only one argument. The only argument must be a pointer to one
// of the events, or an interface{} which would accept all events.
//
// AddHandler would panic if the handler is invalid.
//
//    s.AddHandler(func(m *gateway.MessageCreateEvent) {
//         log.Println(m.Author.Username, "said", m.Content)
//    })
//
package handler

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

type Handler struct {
	// Synchronous controls whether to spawn each event handler in its own
	// goroutine. Default false (meaning goroutines are spawned).
	Synchronous bool

	handlers map[uint64]handler
	horders  []uint64
	hserial  uint64
	hmutex   sync.RWMutex
}

func New() *Handler {
	return &Handler{
		handlers: map[uint64]handler{},
	}
}

// Call calls all handlers with the given event. This is an internal method; use
// with care.
func (h *Handler) Call(ev interface{}) {
	var evV = reflect.ValueOf(ev)
	var evT = evV.Type()

	h.hmutex.RLock()
	defer h.hmutex.RUnlock()

	for _, order := range h.horders {
		handler, ok := h.handlers[order]
		if !ok {
			// This shouldn't ever happen, but we're adding this just in case.
			continue
		}

		if handler.not(evT) {
			continue
		}

		if h.Synchronous {
			handler.call(evV)
		} else {
			go handler.call(evV)
		}
	}
}

// CallDirect is the same as Call, but only calls those event handlers that
// listen for this specific event, i.e. that aren't interface handlers.
func (h *Handler) CallDirect(ev interface{}) {
	var evV = reflect.ValueOf(ev)
	var evT = evV.Type()

	h.hmutex.RLock()
	defer h.hmutex.RUnlock()

	for _, order := range h.horders {
		handler, ok := h.handlers[order]
		if !ok {
			// This shouldn't ever happen, but we're adding this just in case.
			continue
		}

		if evT != handler.event {
			continue
		}

		if h.Synchronous {
			handler.call(evV)
		} else {
			go handler.call(evV)
		}
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
// Function
//
// A handler can be a function with a single argument that is the expected event
// type. It must not have any returns or any other number of arguments.
//
// Channel
//
// A handler can also be a channel. The underlying type that the channel wraps
// around will be the event type. As such, the type rules are the same as
// function handlers.
//
// Keep in mind that the user must NOT close the channel. In fact, the channel
// should not be closed at all. The caller function WILL PANIC if the channel is
// closed!
func (h *Handler) AddHandler(handler interface{}) (rm func()) {
	rm, err := h.addHandler(handler)
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

	return h.addHandler(handler)
}

func (h *Handler) addHandler(fn interface{}) (rm func(), err error) {
	// Reflect the handler
	r, err := newHandler(fn)
	if err != nil {
		return nil, errors.Wrap(err, "handler reflect failed")
	}

	h.hmutex.Lock()
	defer h.hmutex.Unlock()

	// Get the current counter value and increment the counter:
	serial := h.hserial
	h.hserial++

	// Create a map if there's none:
	if h.handlers == nil {
		h.handlers = map[uint64]handler{}
	}

	// Use the serial for the map:
	h.handlers[serial] = *r

	// Append the serial into the list of keys:
	h.horders = append(h.horders, serial)

	return func() {
		h.hmutex.Lock()
		defer h.hmutex.Unlock()

		// Delete the handler from the map:
		delete(h.handlers, serial)

		// Delete the key from the orders slice:
		for i, order := range h.horders {
			if order == serial {
				h.horders = append(h.horders[:i], h.horders[i+1:]...)
				break
			}
		}
	}, nil
}

type handler struct {
	event    reflect.Type // underlying type; arg0 or chan underlying type
	callback reflect.Value
	isChan   bool
	isIface  bool
}

// newHandler reflects either a channel or a function into a handler. A function
// must only have a single argument being the event and no return, and a channel
// must have the event type as the underlying type.
func newHandler(unknown interface{}) (*handler, error) {
	fnV := reflect.ValueOf(unknown)
	fnT := fnV.Type()

	// underlying event type
	var argT reflect.Type
	var isch bool

	switch fnT.Kind() {
	case reflect.Func:
		if fnT.NumIn() != 1 {
			return nil, errors.New("function can only accept 1 event as argument")
		}

		if fnT.NumOut() > 0 {
			return nil, errors.New("function can't accept returns")
		}

		argT = fnT.In(0)

	case reflect.Chan:
		argT = fnT.Elem()
		isch = true

	default:
		return nil, errors.New("given interface is not a function or channel")
	}

	var kind = argT.Kind()

	// Accept either pointer type or interface{} type
	if kind != reflect.Ptr && kind != reflect.Interface {
		return nil, errors.New("first argument is not pointer")
	}

	return &handler{
		event:    argT,
		callback: fnV,
		isChan:   isch,
		isIface:  kind == reflect.Interface,
	}, nil
}

func (h handler) not(event reflect.Type) bool {
	if h.isIface {
		return !event.Implements(h.event)
	}

	return h.event != event
}

func (h handler) call(event reflect.Value) {
	if h.isChan {
		h.callback.Send(event)
	} else {
		h.callback.Call([]reflect.Value{event})
	}
}
