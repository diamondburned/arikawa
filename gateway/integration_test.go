// +build integration

package gateway

import (
	"log"
	"os"
	"testing"
)

func TestIntegration(t *testing.T) {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	WSError = func(err error) {
		log.Println(err)
	}

	var gateway *Gateway

	// NewGateway should call Start for us.
	g, err := NewGateway(token)
	if err != nil {
		t.Fatal("Failed to make a Gateway:", err)
	}
	gateway = g

	ready, ok := (<-gateway.Events).(*ReadyEvent)
	if !ok {
		t.Fatal("Event received is not of type Ready:", ready)
	}

	if gateway.SessionID == "" {
		t.Fatal("Session ID is empty")
	}

	log.Println("Bot's username is", ready.User.Username)

	// Try and reconnect
	if err := gateway.Reconnect(); err != nil {
		t.Fatal("Failed to reconnect:", err)
	}

	/* TODO: this isn't true, as Discord would keep sending Invalid Sessions.
	resumed, ok := (<-gateway.Events).(*ResumedEvent)
	if !ok {
		t.Fatal("Event received is not of type Resumed:", resumed)
	}
	*/

	ready, ok = (<-gateway.Events).(*ReadyEvent)
	if !ok {
		t.Fatal("Event received is not of type Ready:", ready)
	}
}
