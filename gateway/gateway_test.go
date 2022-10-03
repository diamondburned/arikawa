package gateway

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

var doLogOnce sync.Once

func doLog() {
	doLogOnce.Do(func() {
		if testing.Verbose() {
			ws.WSDebug = func(v ...interface{}) {
				log.Println(append([]interface{}{"Debug:"}, v...)...)
			}
		}
	})
}

func TestURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	u, err := URL(ctx)
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
	doLog()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	g, err := New(ctx, "bad token")
	if err != nil {
		t.Fatal("failed to make a Gateway:", err)
	}

	assertIsClose := func(err error) {
		if err == nil {
			t.Fatal("unexpected nil error")
		}

		// 4004 Authentication Failed.
		if !strings.Contains(err.Error(), "4004") {
			t.Fatal("unexpected error:", err)
		}
	}

	for op := range g.Connect(ctx) {
		if op.Data == nil {
			// This shouldn't happen; the loop should've broken out.
			t.Fatal("nil event received")
		}

		switch data := op.Data.(type) {
		case *ws.CloseEvent:
			assertIsClose(data)
		case *ws.BackgroundErrorEvent:
			t.Error("gateway error:", data)
		case *HelloEvent:
			t.Log("got Hello")
		case *InvalidSessionEvent:
			t.Log("got InvalidSession")
		default:
			t.Errorf("got unexpected event %#v", data)
		}
	}

	assertIsClose(g.LastError())
}

func TestIntegration(t *testing.T) {
	doLog()

	config := testenv.Must(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	// NewGateway should call Start for us.
	g, err := NewWithIntents(ctx, "Bot "+config.BotToken, IntentGuilds)
	if err != nil {
		t.Fatal("failed to make a Gateway:", err)
	}

	gatewayOpenAndSpin(t, ctx, g)
	cancel()
}

func TestReuseGateway(t *testing.T) {
	doLog()

	config := testenv.Must(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	// NewGateway should call Start for us.
	g, err := NewWithIntents(ctx, "Bot "+config.BotToken, IntentGuilds)
	if err != nil {
		t.Fatal("failed to make a Gateway:", err)
	}

	// Reuse this 3 times.
	for i := 0; i < 3; i++ {
		cctx, cancel := context.WithCancel(ctx)
		gatewayOpenAndSpin(t, cctx, g)
		cancel()
	}
}

func gatewayOpenAndSpin(t *testing.T, ctx context.Context, g *Gateway) {
	ch := g.Connect(ctx)

	var reconnected bool
	reconnect := func() {
		if !reconnected {
			reconnected = true
			g.gateway.QueueReconnect()
		}
	}

	for op := range ch {
		if op.Data == nil {
			// This shouldn't happen; the loop should've broken out.
			t.Fatal("nil event received")
		}

		switch data := op.Data.(type) {
		case *ReadyEvent:
			t.Log("got Ready")
			if g.state.SessionID != data.SessionID {
				t.Fatal("missing SessionID")
			}
			if strconv.Itoa(data.Version) != api.Version {
				t.Fatalf("mismatch version, got %d", data.Version)
			}
			log.Println("Bot's username is", data.User.Username)
			reconnect()
		case *ResumedEvent:
			t.Log("got Resumed, test done")
			return
		case *HelloEvent:
			t.Log("got Hello")
		case *ws.BackgroundErrorEvent:
			t.Error("gateway error:", data)
		default:
			t.Logf("got event %T", data)
		}
	}
}
