package gateway

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/internal/heart"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
)

func init() {
	wsutil.WSDebug = func(v ...interface{}) {
		log.Println(append([]interface{}{"Debug:"}, v...)...)
	}
	heart.Debug = func(v ...interface{}) {
		log.Println(append([]interface{}{"Heart:"}, v...)...)
	}
}

func TestURL(t *testing.T) {
	u, err := URL()
	if err != nil {
		t.Fatal("failed to get gateway URL:", err)
	}

	if u == "" {
		t.Fatal("gateway URL is empty")
	}

	if !strings.HasPrefix(u, "wss://") {
		t.Fatal("gatewayURL is invalid:", u)
	}
}

func TestInvalidToken(t *testing.T) {
	g, err := NewGateway("bad token")
	if err != nil {
		t.Fatal("failed to make a Gateway:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err = g.Open(ctx); err == nil {
		t.Fatal("unexpected success while opening with a bad token.")
	}

	// 4004 Authentication Failed.
	if !strings.Contains(err.Error(), "4004") {
		t.Fatal("unexpected error:", err)
	}
}

func TestIntegration(t *testing.T) {
	config := testenv.Must(t)

	var gateway *Gateway

	// NewGateway should call Start for us.
	g, err := NewGateway("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("failed to make a Gateway:", err)
	}
	g.AddIntents(IntentGuilds)
	g.AfterClose = func(err error) {
		t.Log("closed.")
	}
	g.ErrorLog = func(err error) {
		t.Log("gateway error:", err)
	}
	gateway = g

	gotimeout(t, func(ctx context.Context) {
		if err := g.Open(ctx); err != nil {
			t.Fatal("failed to authenticate with Discord:", err)
		}
	})

	ev := wait(t, gateway.Events)
	ready, ok := ev.(*ReadyEvent)
	if !ok {
		t.Fatal("event received is not of type Ready:", ev)
	}

	if gateway.SessionID() == "" {
		t.Fatal("session ID is empty")
	}

	log.Println("Bot's username is", ready.User.Username)

	// Send a faster heartbeat every second for testing.
	g.PacerLoop.SetPace(time.Second)

	// Sleep past the rate limiter before reconnecting:
	time.Sleep(5 * time.Second)

	gotimeout(t, func(ctx context.Context) {
		g.ErrorLog = func(err error) {
			t.Error("unexpected error while reconnecting:", err)
		}

		if err := gateway.ReconnectCtx(ctx); err != nil {
			t.Error("failed to reconnect Gateway:", err)
		}
	})

	g.ErrorLog = func(err error) { t.Log("warning:", err) }

	// Wait for the desired event:
	gotimeout(t, func(context.Context) {
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
		t.Fatal("failed to close Gateway:", err)
	}
}

func wait(t *testing.T, evCh chan interface{}) interface{} {
	select {
	case ev := <-evCh:
		return ev
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for event")
		return nil
	}
}

func gotimeout(t *testing.T, fn func(context.Context)) {
	t.Helper()

	// Try and reconnect for 20 seconds maximum.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var done = make(chan struct{})
	go func() {
		fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		t.Fatal("timed out waiting for function.")
	case <-done:
		return
	}
}
