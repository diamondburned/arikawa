package session

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
)

func TestSession(t *testing.T) {
	attempts := 1
	timeout := 15 * time.Second

	if !testing.Short() {
		attempts = 5
		timeout = time.Minute // 5s-10s each reconnection
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	env := testenv.Must(t)

	readyCh := make(chan *gateway.ReadyEvent, 1)

	s := NewWithIntents(env.BotToken, gateway.IntentGuilds)
	s.AddHandler(readyCh)

	for i := 0; i < attempts; i++ {
		if err := s.Open(ctx); err != nil {
			t.Fatal("failed to open:", err)
		}

		if ready, ok := <-readyCh; !ok {
			t.Fatal("ready not received")
		} else {
			now := time.Now()
			t.Logf("%s: logged in as %s", now.Format(time.StampMilli), ready.User.Username)
		}

		if err := s.Close(); err != nil {
			t.Fatal("failed to close:", err)
		}

		// Hold for an additional one second.
		time.Sleep(time.Second)
	}
}
