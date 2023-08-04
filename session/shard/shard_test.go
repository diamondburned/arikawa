package shard

import (
	"context"
	"testing"
	"time"

	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/internal/testenv"
	"libdb.so/arikawa/v4/session"
)

func TestSharding(t *testing.T) {
	env := testenv.Must(t)

	data := gateway.DefaultIdentifyCommand("Bot " + env.BotToken)
	data.Shard = &gateway.Shard{0, env.ShardCount}

	readyCh := make(chan *gateway.ReadyEvent)

	m, err := NewIdentifiedManager(data, NewSessionShard(
		func(m *Manager, s *session.Session) {
			now := time.Now().Format(time.StampMilli)
			t.Log(now, "initializing shard")

			s.AddIntents(gateway.IntentGuilds)
			s.AddHandler(readyCh)
			s.AddHandler(func(err error) {
				t.Error("unexpected error:", err)
			})
		},
	))
	if err != nil {
		t.Fatal("failed to make shard manager:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
