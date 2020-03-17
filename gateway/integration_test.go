// +build integration

package gateway

import (
	"log"
	"os"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestInvalidToken(t *testing.T) {
	g, err := NewGateway("bad token")
	if err != nil {
		t.Fatal("Failed to make a Gateway:", err)
	}

	err = g.Open()
	if err == nil {
		t.Fatal("Unexpected success while opening with a bad token.")
	}

	// 4004 Authentication Failed.
	if websocket.CloseStatus(err) == 4004 {
		return
	}

	t.Fatal("Unexpected error:", err)
}

func TestIntegration(t *testing.T) {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	WSError = func(err error) {
		t.Fatal(err)
	}

	var gateway *Gateway

	// NewGateway should call Start for us.
	g, err := NewGateway("Bot " + token)
	if err != nil {
		t.Fatal("Failed to make a Gateway:", err)
	}
	gateway = g

	if err := g.Open(); err != nil {
		t.Fatal("Failed to authenticate with Discord:", err)
	}

	ev := wait(t, gateway.Events)
	ready, ok := ev.(*ReadyEvent)
	if !ok {
		t.Fatal("Event received is not of type Ready:", ev)
	}

	if gateway.SessionID == "" {
		t.Fatal("Session ID is empty")
	}

	log.Println("Bot's username is", ready.User.Username)

	// Try and reconnect
	if err := gateway.Reconnect(); err != nil {
		t.Fatal("Failed to reconnect:", err)
	}

	timeout := time.After(10 * time.Second)

Main:
	for {
		select {
		case ev := <-gateway.Events:
			switch ev.(type) {
			case *ResumedEvent, *ReadyEvent:
				break Main
			}
		case <-timeout:
			t.Fatal("Timed out waiting for ResumedEvent")
		}
	}

	if err := g.Close(); err != nil {
		t.Fatal("Failed to close Gateway:", err)
	}
}

func wait(t *testing.T, evCh chan interface{}) interface{} {
	select {
	case ev := <-evCh:
		return ev
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
		return nil
	}
}
