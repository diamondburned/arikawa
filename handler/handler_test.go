// +build unit

package handler

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/gateway"
)

func TestCall(t *testing.T) {
	var results = make(chan string)

	h := &Handler{
		handlers: map[uint64]handler{},
	}

	// Add handler test
	rm := h.AddHandler(func(m *gateway.MessageCreateEvent) {
		results <- m.Content
	})

	go h.Call(&gateway.MessageCreateEvent{
		Content: "test",
	})

	if r := <-results; r != "test" {
		t.Fatal("Returned results is wrong:", r)
	}

	// Remove handler test
	rm()

	go h.Call(&gateway.MessageCreateEvent{
		Content: "test",
	})

	select {
	case <-results:
		t.Fatal("Unexpected results")
	case <-time.After(time.Millisecond):
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

	h, err := reflectFn(func(m *gateway.MessageCreateEvent) {
		results <- m.Content
	})
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = &gateway.MessageCreateEvent{
		Content: result,
	}

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

func TestHandlerInterface(t *testing.T) {
	var results = make(chan interface{})

	h, err := reflectFn(func(m interface{}) {
		results <- m
	})
	if err != nil {
		t.Fatal(err)
	}

	const result = "Hime Arikawa"
	var msg = &gateway.MessageCreateEvent{
		Content: result,
	}

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

func BenchmarkReflect(b *testing.B) {
	h, err := reflectFn(func(m *gateway.MessageCreateEvent) {})
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
