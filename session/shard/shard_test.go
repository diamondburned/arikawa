package shard_test

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/session/shard"
)

func TestSharding(t *testing.T) {
	env := testenv.Must(t)

	data := gateway.DefaultIdentifyCommand("Bot " + env.BotToken)
	data.Shard = &gateway.Shard{0, env.ShardCount}

	readyCh := make(chan *gateway.ReadyEvent)

	m, err := shard.NewIdentifiedManager(data, shard.NewSessionShard(
		func(m *shard.Manager, s *session.Session) {
			now := time.Now().Format(time.StampMilli)
			t.Log(now, "initializing shard")

			s.AddIntents(gateway.IntentGuilds)
			s.AddHandler(readyCh)
			s.AddHandler(func(err error) {
				t.Log(err)
			})
		},
	))
	if err != nil {
		t.Fatal("failed to make shard manager:", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Minute*time.Duration(env.ShardCount))
	defer cancel()

	openDone := make(chan struct{})
	go func() {
		defer close(openDone)
		if err := m.Open(ctx); err != nil {
			t.Error("failed to open:", err)
		}
	}()
	t.Cleanup(func() { m.Close() })

shardLoop:
	for i := 0; i < env.ShardCount; i++ {
		select {
		case ready := <-readyCh:
			now := time.Now().Format(time.StampMilli)
			t.Log(now, "shard", ready.Shard.ShardID(), "is ready out of", env.ShardCount)
		case <-ctx.Done():
			t.Error("test expired, got", i, "shards")
			break shardLoop
		}
	}

	select {
	case <-openDone:
		t.Log("all shards opened")
	case <-ctx.Done():
		t.Error("test expired")
	}

	if err := m.Close(); err != nil {
		t.Error("failed to close:", err)
	}
}
