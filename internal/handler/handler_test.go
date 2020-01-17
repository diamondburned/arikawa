package handler

import (
	"reflect"
	"testing"

	"github.com/diamondburned/arikawa/gateway"
)

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
