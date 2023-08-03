package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/handler"
)

func newMessage(content string) *gateway.MessageCreateEvent {
	return &gateway.MessageCreateEvent{
		Message: discord.Message{Content: content},
	}
}

func TestHandlers(t *testing.T) {
	h := handler.New[gateway.Event]()

	t.Run("HandleCallback", func(t *testing.T) {
		var dispatched bool
		ch := make(chan gateway.Event, 1)
		rm := h.HandleCallback(func(ev gateway.Event) {
			time.Sleep(10 * time.Millisecond)
			dispatched = true
			ch <- ev
		})

		ev := newMessage("hime arikawa")
		h.Dispatch(ev)
		assert.Equal(t, dispatched, false, "callback dispatched too early")
		assert.Equal(t, chOnce(t, ch), gateway.Event(ev))

		rm()
		dispatched = false
		h.Dispatch(ev)
		assert.Equal(t, dispatched, false, "callback dispatched after removal")
	})

	t.Run("HandleSynchronousCallback", func(t *testing.T) {
		var dispatched bool
		ch := make(chan gateway.Event, 1)
		rm := h.HandleSynchronousCallback(func(ev gateway.Event) {
			time.Sleep(10 * time.Millisecond)
			dispatched = true
			ch <- ev
		})

		ev := newMessage("hime arikawa")
		h.Dispatch(ev)
		assert.Equal(t, dispatched, true, "callback not dispatched")
		assert.Equal(t, chOnce(t, ch), gateway.Event(ev))

		rm()
		dispatched = false
		h.Dispatch(ev)
		assert.Equal(t, dispatched, false, "callback dispatched after removal")
	})

	addChannelFuncs := []struct {
		name string
		add  func(chan<- gateway.Event) func()
	}{
		{"HandleChannel", h.HandleChannel},
		{"HandleBlockingChannel", h.HandleBlockingChannel},
	}

	for _, test := range addChannelFuncs {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan gateway.Event, 1)
			rm := test.add(ch)

			ev := newMessage("hime arikawa")
			h.Dispatch(ev)
			assert.Equal(t, chOnce(t, ch), gateway.Event(ev))

			rm()
			h.Dispatch(ev)
			chNone(t, ch)
		})
	}
}

func BenchmarkHandlerAddRemove(b *testing.B) {
	h := handler.New[gateway.Event]()
	for i := 0; i < b.N; i++ {
		rm := h.HandleCallback(func(ev gateway.Event) {})
		rm()
	}
}

func TestAdd(t *testing.T) {
	h := handler.New[gateway.Event]()

	ch := make(chan *gateway.MessageCreateEvent, 1)
	handler.Add[gateway.Event](h, func(ev *gateway.MessageCreateEvent) { ch <- ev })

	ev := newMessage("hime arikawa")
	h.Dispatch(ev)
	assert.Equal(t, chOnce(t, ch), ev)

	h.Dispatch(&gateway.ReadyEvent{})
	chNone(t, ch)
}

func BenchmarkAddLatency(b *testing.B) {
	h := handler.New[gateway.Event]()
	ev := newMessage("hime arikawa")
	ch := make(chan *gateway.MessageCreateEvent, 1)
	handler.Add[gateway.Event](h, func(ev *gateway.MessageCreateEvent) { ch <- ev })

	for i := 0; i < b.N; i++ {
		h.Dispatch(ev)
		<-ch
	}
}

func TestAddSynchronous(t *testing.T) {
	h := handler.New[gateway.Event]()

	ch := make(chan *gateway.MessageCreateEvent, 1)
	handler.AddSynchronous[gateway.Event](h, func(ev *gateway.MessageCreateEvent) { ch <- ev })

	ev := newMessage("hime arikawa")
	h.Dispatch(ev)
	assert.Equal(t, chOnce(t, ch), ev)

	h.Dispatch(&gateway.ReadyEvent{})
	chNone(t, ch)
}

func BenchmarkAddSynchronousLatency(b *testing.B) {
	h := handler.New[gateway.Event]()
	ev := newMessage("hime arikawa")
	ch := make(chan *gateway.MessageCreateEvent, 1)
	handler.AddSynchronous[gateway.Event](h, func(ev *gateway.MessageCreateEvent) { ch <- ev })

	for i := 0; i < b.N; i++ {
		h.Dispatch(ev)
		<-ch
	}
}

func TestExpect(t *testing.T) {
	events := []gateway.Event{
		newMessage("hello world"),
		newMessage("hime arikawa"),
		&gateway.ReadyEvent{},
	}

	filter := func(ev *gateway.MessageCreateEvent) bool {
		return ev.Content == "hime arikawa"
	}

	want := events[1]

	h := handler.New[gateway.Event]()
	dispatchAll := func() {
		for _, ev := range events {
			h.Dispatch(ev)
		}
	}

	t.Run("Expect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		recv := handler.Expect[gateway.Event](h, filter)
		go dispatchAll()

		v, err := recv(ctx)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		assert.Equal(t, gateway.Event(v), want)
	})

	t.Run("ExpectCh", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		recvCh := handler.ExpectCh[gateway.Event](ctx, h, filter)
		go dispatchAll()
		go dispatchAll() // ensure we can get multiple events

		for i := 0; i < 2; i++ {
			select {
			case v := <-recvCh:
				assert.Equal(t, gateway.Event(v), want)
			case <-ctx.Done():
				t.Fatal("timed out")
			}
		}
	})
}

func chOnce[T any](t *testing.T, ch <-chan T) T {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case v := <-ch:
		return v
	case <-timer.C:
		t.Fatal("channel timed out")
		panic("unreachable")
	}
}

func chNone[T any](t *testing.T, ch <-chan T) {
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()

	select {
	case v := <-ch:
		t.Fatal("unexpected value:", v)
	case <-timer.C:
	}
}
