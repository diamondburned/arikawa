package handler

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

func newMessage(content string) *gateway.MessageCreateEvent {
	return &gateway.MessageCreateEvent{
		Message: discord.Message{Content: content},
	}
}

func TestCall(t *testing.T) {
	var results = make(chan string)

	h := &Handler{}

	// Add handler test
	rm := h.AddHandler(func(m *gateway.MessageCreateEvent) {
		results <- m.Content
	})

	go h.Call(newMessage("hime arikawa"))

	if r := <-results; r != "hime arikawa" {
		t.Fatal("Returned results is wrong:", r)
	}

	// Delete handler test
	rm()

	go h.Call(newMessage("astolfo"))

	select {
	case <-results:
		t.Fatal("Unexpected results")
	case <-time.After(5 * time.Millisecond):
		break
	}

	// Invalid type test
	_, err := h.AddHandlerCheck("this should panic")
	if err == nil {
		t.Fatal("No errors found")
	}

	// We don't do anything with the returned callback, as there's none.

	if !strings.Contains(err.Error(), "given interface is not a function") {
		t.Fatal("Unexpected error:", err)
	}
}

func TestHandler(t *testing.T) {
	var results = make(chan string)

	h, err := newHandler(func(m *gateway.MessageCreateEvent) {
		results <- m.Content
	})
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = newMessage(result)

	var msgV = reflect.ValueOf(msg)
	var msgT = msgV.Type()

	if h.not(msgT) {
		t.Fatal("Event type mismatch")
	}

	go h.call(msgV)

	if results := <-results; results != result {
		t.Fatal("Unexpected results:", results)
	}
}

func TestHandlerChan(t *testing.T) {
	var results = make(chan *gateway.MessageCreateEvent)

	h, err := newHandler(results)
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = newMessage(result)

	var msgV = reflect.ValueOf(msg)
	var msgT = msgV.Type()

	if h.not(msgT) {
		t.Fatal("Event type mismatch")
	}

	go h.call(msgV)

	if results := <-results; results.Content != result {
		t.Fatal("Unexpected results:", results)
	}
}

func TestHandlerChanCancel(t *testing.T) {
	// Never receive from this channel. It is important that this channel is
	// unbuffered.
	var results = make(chan *gateway.MessageCreateEvent)

	h, err := newHandler(results)
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = newMessage(result)

	var msgV = reflect.ValueOf(msg)
	var msgT = msgV.Type()

	if h.not(msgT) {
		t.Fatal("Event type mismatch")
	}

	// Channel that waits for call() to die.
	die := make(chan struct{})

	// Call in a goroutine, which would trigger a close.
	go func() { h.call(msgV); die <- struct{}{} }()

	// Call the cleanup function, which should stop the send.
	h.cleanup()

	// Check if we still have things being sent.
	select {
	case <-die:
		// pass
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for call routine to die.")
	}

	// Check if we still receive something.
	select {
	case <-results:
		t.Fatal("Unexpected results received.")
	default:
		// pass
	}
}

func TestHandlerInterface(t *testing.T) {
	var results = make(chan interface{})

	h, err := newHandler(func(m interface{}) {
		results <- m
	})
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = newMessage(result)

	var msgV = reflect.ValueOf(msg)
	var msgT = msgV.Type()

	if h.not(msgT) {
		t.Fatal("Event type mismatch")
	}

	go h.call(msgV)
	recv := <-results

	if msg, ok := recv.(*gateway.MessageCreateEvent); ok {
		if msg.Content == result {
			return
		}

		t.Fatal("Content mismatch:", msg.Content)
	}

	t.Fatal("Assertion failed:", recv)
}

func TestHandlerWaitFor(t *testing.T) {
	inc := make(chan interface{}, 1)

	h := New()

	wanted := &gateway.TypingStartEvent{
		ChannelID: 123456,
	}

	evs := []interface{}{
		&gateway.TypingStartEvent{},
		&gateway.MessageCreateEvent{},
		&gateway.ChannelDeleteEvent{},
		wanted,
	}

	go func() {
		inc <- h.WaitFor(context.Background(), func(v interface{}) bool {
			tp, ok := v.(*gateway.TypingStartEvent)
			if !ok {
				return false
			}

			return tp.ChannelID == wanted.ChannelID
		})
	}()

	// Wait for WaitFor to add its handler:
	time.Sleep(time.Millisecond)

	for _, ev := range evs {
		h.Call(ev)
	}

	recv := <-inc
	if recv != wanted {
		t.Fatal("Unexpected receive:", recv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	// Test timeout
	v := h.WaitFor(ctx, func(v interface{}) bool {
		return false
	})

	if v != nil {
		t.Fatal("Unexpected value:", v)
	}
}

func TestHandlerChanFor(t *testing.T) {
	h := New()

	wanted := &gateway.TypingStartEvent{
		ChannelID: 123456,
	}

	evs := []interface{}{
		&gateway.TypingStartEvent{},
		&gateway.MessageCreateEvent{},
		&gateway.ChannelDeleteEvent{},
		wanted,
	}

	inc, cancel := h.ChanFor(func(v interface{}) bool {
		tp, ok := v.(*gateway.TypingStartEvent)
		if !ok {
			return false
		}

		return tp.ChannelID == wanted.ChannelID
	})
	defer cancel()

	for _, ev := range evs {
		h.Call(ev)
	}

	recv := <-inc
	if recv != wanted {
		t.Fatal("Unexpected receive:", recv)
	}
}

func BenchmarkReflect(b *testing.B) {
	h, err := newHandler(func(m *gateway.MessageCreateEvent) {})
	if err != nil {
		b.Fatal(err)
	}

	var msg = &gateway.MessageCreateEvent{}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		var msgV = reflect.ValueOf(msg)
		var msgT = msgV.Type()

		if h.not(msgT) {
			b.Fatal("Event type mismatch")
		}

		h.call(msgV)
	}
}
