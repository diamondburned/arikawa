// +build perseverance

package gateway

import (
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/internal/testenv"
)

func TestPerseverance(t *testing.T) {
	t.Parallel()

	config := testenv.Must(t)

	g, err := NewGateway("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("failed to make the gateway:", err)
	}
	g.AddIntents(IntentGuilds)

	if err := g.Open(); err != nil {
		t.Fatal("failed to open the gateway:", err)
	}

	timeout := make(chan struct{}, 1)

	// Automatically close the gateway after set duration.
	time.AfterFunc(testenv.PerseveranceTime, func() {
		t.Log("Perserverence test finshed. Closing gateway.")
		timeout <- struct{}{}

		if err := g.Close(); err != nil {
			t.Error("failed to close gateway:", err)
		}
	})

	// Spin on events.
	for ev := range g.Events {
		t.Logf("Received event %T.", ev)
	}

	// Exit gracefully if we have not.
	select {
	case <-timeout:
		return
	default:
	}

	if err := g.Close(); err != nil {
		t.Fatal("failed to clean up gateway after fail:", err)
	}

	t.Fatal("Test failed before timeout.")
}
