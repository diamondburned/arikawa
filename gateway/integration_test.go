// +build integration

package gateway

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v2/internal/heart"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
)

func init() {
	wsutil.WSDebug = func(v ...interface{}) {
		log.Println(append([]interface{}{"Debug:"}, v...)...)
	}
	heart.Debug = func(v ...interface{}) {
		log.Println(append([]interface{}{"Heart:"}, v...)...)
	}
}

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
	if strings.Contains(err.Error(), "4004") {
		return
	}

	t.Fatal("Unexpected error:", err)
}

func TestIntegration(t *testing.T) {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	wsutil.WSError = func(err error) {
		t.Fatal(err)
	}

	var gateway *Gateway

	// NewGateway should call Start for us.
	g, err := NewGateway("Bot " + token)
	if err != nil {
		t.Fatal("Failed to make a Gateway:", err)
	}
	g.AfterClose = func(err error) {
		log.Println("Closed.")
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

	// Sleep past the rate limiter before reconnecting:
	time.Sleep(5 * time.Second)

	gotimeout(t, func() {
		// Try and reconnect for 20 seconds maximum.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := gateway.ReconnectCtx(ctx); err != nil {
			t.Fatal("Unexpected error while reconnecting:", err)
		}
	})

	// Wait for the desired event:
	gotimeout(t, func() {
		for ev := range gateway.Events {
			switch ev.(type) {
			// Accept only a Resumed event.
			case *ResumedEvent:
				return // exit
			case *ReadyEvent:
				t.Fatal("Ready event received instead of Resumed.")
			}
		}
	})

	if err := g.Close(); err != nil {
		t.Fatal("Failed to close Gateway:", err)
	}
}

func wait(t *testing.T, evCh chan interface{}) interface{} {
	select {
	case ev := <-evCh:
		return ev
	case <-time.After(20 * time.Second):
		t.Fatal("Timed out waiting for event")
		return nil
	}
}

func gotimeout(t *testing.T, fn func()) {
	t.Helper()

	var done = make(chan struct{})
	go func() {
		fn()
		done <- struct{}{}
	}()

	select {
	case <-time.After(20 * time.Second):
		t.Fatal("Timed out waiting for function.")
	case <-done:
		return
	}
}
