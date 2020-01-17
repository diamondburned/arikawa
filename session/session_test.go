package session

import (
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/gateway"
)

func TestSessionCall(t *testing.T) {
	var results = make(chan string)

	s := &Session{
		handlers: map[uint64]handler{},
	}

	// Add handler test
	rm := s.AddHandler(func(m *gateway.MessageCreateEvent) {
		results <- m.Content
	})

	go s.call(&gateway.MessageCreateEvent{
		Content: "test",
	})

	if r := <-results; r != "test" {
		t.Fatal("Returned results is wrong:", r)
	}

	// Remove handler test
	rm()

	go s.call(&gateway.MessageCreateEvent{
		Content: "test",
	})

	select {
	case <-results:
		t.Fatal("Unexpected results")
	case <-time.After(time.Millisecond):
		break
	}

	// Invalid type test
	rm, err := s.AddHandlerCheck("this should panic")
	if err == nil {
		t.Fatal("No errors found")
	}
	defer rm()

	if !strings.Contains(err.Error(), "given interface is not a function") {
		t.Fatal("Unexpected error:", err)
	}
}
