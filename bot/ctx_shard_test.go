package bot

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/state"
)

type shardedBot struct {
	Ctx *Context

	readyCh chan *gateway.ReadyEvent
}

func (bot *shardedBot) OnReady(r *gateway.ReadyEvent) {
	bot.readyCh <- r
}

func TestSharding(t *testing.T) {
	env := testenv.Must(t)

	data := gateway.DefaultIdentifyData("Bot " + env.BotToken)
	data.Shard = &gateway.Shard{0, env.ShardCount}

	readyCh := make(chan *gateway.ReadyEvent)

	newShard := NewShardFunc(func(s *state.State) (*Context, error) {
		b, err := New(s, &shardedBot{nil, readyCh})
		if err != nil {
			return nil, err
		}

		b.AddIntents(gateway.IntentGuilds)
		return b, nil
	})

	m, err := shard.NewIdentifiedManager(data, newShard)
	if err != nil {
		t.Fatal("failed to make shard manager:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go func() {
		// Timeout
		if err := m.Open(ctx); err != nil {
			t.Error("failed to open:", err)
			cancel()
		}

		t.Cleanup(func() {
			if err := m.Close(); err != nil {
				t.Error("failed to close:", err)
				cancel()
			}
		})
	}()

	// Expect 4 Ready events.
	for i := 0; i < env.ShardCount; i++ {
		select {
		case ready := <-readyCh:
			now := time.Now().Format(time.StampMilli)
			t.Log(now, "shard", ready.Shard.ShardID(), "is ready out of", env.ShardCount)
		case <-ctx.Done():
			t.Fatal("test expired, got", i, "shards")
		}
	}
}
