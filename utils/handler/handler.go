// Package handler handles incoming Gateway events. It reflects the function's
// first argument and caches that for use in each event.
//
// Performance
//
// Benchmark results replicated with `go test -bench=.`:
//
//    goos: linux
//    goarch: amd64
//    pkg: libdb.so/arikawa/v4/utils/handler
//    cpu: 12th Gen Intel(R) Core(TM) i5-1240P
//    BenchmarkHandleRemove
//    BenchmarkHandleRemove-16                	2910373	      395.2 ns/op
//    BenchmarkHandleLatency
//    BenchmarkHandleLatency-16               	1622205	      761.9 ns/op
//    BenchmarkHandleSynchronousLatency
//    BenchmarkHandleSynchronousLatency-16    	10508937	      108.2 ns/op
//    PASS
//    ok  	libdb.so/arikawa/v4/utils/handler	4.884s
//
// Usage
//
// handler's usage is mostly similar to Discordgo, in that Addhandler expects a
// function with only one argument or an event channel. For more information,
// refer to Addhandler.
package handler

import (
	"context"
	"sync"
	"sync/atomic"
)

// Dispatcher is an interface for dispatching events.
type Dispatcher[T any] interface {
	// Dispatch dispatches all handlers with the given event. The method blocks
	// until all handlers are done.
	Dispatch(ev T)
}

// Handler is an interface for adding callbacks and channels.
type Handler[T any] interface {
	// HandleCallback adds a callback function that is called on every dispatched
	// event. It returns a function that would remove this handler when called.
	// Callbacks are dispatched in its own goroutine.
	HandleCallback(fn func(T)) (rm func())
	// HandleSynchronousCallback is like AddCallback, but it's called
	// synchronously. Use this only for non-blocking operations such as
	// dispatching to other handlers.
	HandleSynchronousCallback(fn func(T)) (rm func())
	// HandleChannel adds the given channel to receive dispatched events. If the
	// channel is full, the Dispatch caller will block until the channel is
	// available. If a channel is never available, the dispatch goroutine will
	// dangle indefinitely.
	//
	// Keep in mind that the user must NOT close the channel. In fact, the
	// channel should not be closed at all. The caller function WILL PANIC if
	// the channel is closed!
	//
	// When the rm callback that is returned is called, it will also guarantee
	// that all blocking sends will be cancelled. This helps prevent dangling
	// goroutines.
	//
	// Example usage:
	//
	//    ch := make(chan *gateway.MessageCreateEvent)
	//    rm := h.Addhandler(ch)
	//    defer rm()
	//
	//    for ev := range ch {
	//        // do something with ev
	//    }
	//
	HandleChannel(ch chan<- T) (rm func())
	// HandleBlockingChannel is like AddChannel, but the Dispatch caller will
	// block until the channel is available to receive the event. If the
	// channel is never available, the dispatch caller will block indefinitely.
	HandleBlockingChannel(ch chan<- T) (rm func())
}

// Add adds a callback function that is called on every dispatched event to
// the given handler. If the dispatched type does not implement the callback's
// argument type, it is ignored. The callback is dispatched asynchronously.
func Add[HandlerT any, EventT any](h Handler[HandlerT], fn func(EventT)) (rm func()) {
	assertImpl[HandlerT, EventT]()

	return h.HandleSynchronousCallback(func(ev HandlerT) {
		if e, ok := any(ev).(EventT); ok {
			go fn(e)
		}
	})
}

// AddSynchronous is like Add, but the callback is dispatched synchronously.
func AddSynchronous[handlerT any, EventT any](h Handler[handlerT], fn func(EventT)) (rm func()) {
	assertImpl[handlerT, EventT]()

	return h.HandleSynchronousCallback(func(ev handlerT) {
		if e, ok := any(ev).(EventT); ok {
			fn(e)
		}
	})
}

// Expect returns a function that blocks until the given callback returns true,
// and then returns the event. If the context is canceled, it returns false.
func Expect[HandlerT, EventT any](h Handler[HandlerT], fn func(EventT) bool) func(context.Context) (EventT, error) {
	assertImpl[HandlerT, EventT]()

	out := make(chan HandlerT)
	rm := h.HandleChannel(out)

	return func(ctx context.Context) (EventT, error) {
		defer rm()

		for {
			select {
			case <-ctx.Done():
				var z EventT
				return z, ctx.Err()
			case ev := <-out:
				v, ok := any(ev).(EventT)
				if ok && fn(v) {
					return v, nil
				}
			}
		}
	}
}

// ExpectCh is like Expect, but it returns a channel instead. The channel is no
// longer sent to when the context is canceled. Unlike Expect, the returned
// channel can receive multiple events.
func ExpectCh[HandlerT, EventT any](ctx context.Context, h Handler[HandlerT], fn func(EventT) bool) <-chan EventT {
	assertImpl[HandlerT, EventT]()

	evs := make(chan EventT, 1)
	out := make(chan HandlerT, 1)
	rm := h.HandleChannel(out)

	go func() {
		defer rm()

		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-out:
				v, ok := any(ev).(EventT)
				if ok && fn(v) {
					select {
					case <-ctx.Done():
						return
					case evs <- v:
					}
				}
			}
		}
	}()

	return evs
}

// Handlers is a container for command handlers. A zero-value instance is a valid
// instance.
type Handlers[T any] struct {
	mutex   sync.RWMutex
	callers slab[caller[T]] // nil type for interfaces
}

var (
	_ Dispatcher[struct{}] = (*Handlers[struct{}])(nil)
	_ Handler[struct{}]    = (*Handlers[struct{}])(nil)
)

// New constructs a zero-value Handler.
func New[T any]() interface {
	Dispatcher[T]
	Handler[T]
} {
	return &Handlers[T]{callers: newSlab[caller[T]](12)}
}

// Dispatch implements Dispatcher.
func (h *Handlers[T]) Dispatch(ev T) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	h.callers.All(func(c caller[T]) bool {
		c.Call(ev)
		return true
	})
}

// HandleCallback implements Handler.
func (h *Handlers[T]) HandleCallback(Handler func(T)) (rm func()) {
	return h.add(callback[T]{fn: Handler, async: true})
}

// HandleSynchronousCallback implements Handler.
func (h *Handlers[T]) HandleSynchronousCallback(Handler func(T)) (rm func()) {
	return h.add(callback[T]{fn: Handler})
}

// HandleChannel implements Handler.
func (h *Handlers[T]) HandleChannel(ch chan<- T) (rm func()) {
	return h.add(channel[T]{ch: ch, close: make(chan struct{}), async: true})
}

// HandleBlockingChannel implements Handler.
func (h *Handlers[T]) HandleBlockingChannel(ch chan<- T) (rm func()) {
	return h.add(channel[T]{ch: ch, close: make(chan struct{})})
}

func (h *Handlers[T]) add(c caller[T]) (rm func()) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	i := h.callers.Put(c)
	var gone atomic.Bool

	return func() {
		if !gone.CompareAndSwap(false, true) {
			return
		}

		h.mutex.Lock()
		c := h.callers.Pop(i)
		h.mutex.Unlock()
		c.Close()
	}
}

type caller[T any] interface {
	Call(T)
	Close()
}

var (
	_ caller[struct{}] = callback[struct{}]{}
	_ caller[struct{}] = channel[struct{}]{}
)

type callback[T any] struct {
	fn    func(T)
	async bool
}

func (c callback[T]) Call(v T) {
	if c.async {
		go c.fn(v)
	} else {
		c.fn(v)
	}
}

func (c callback[T]) Close() {}

type channel[T any] struct {
	ch    chan<- T
	close chan struct{}
	async bool
}

func (c channel[T]) Call(v T) {
	select {
	case <-c.close:
		return
	default:
	}

	if c.async {
		go func() {
			select {
			case c.ch <- v:
			case <-c.close:
			}
		}()
	} else {
		select {
		case c.ch <- v:
		case <-c.close:
		}
	}
}

func (c channel[T]) Close() {
	select {
	case <-c.close:
	default:
		close(c.close)
	}
}
