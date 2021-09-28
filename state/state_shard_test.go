package state

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/session/shard"
)

func TestSharding(t *testing.T) {
	env := testenv.Must(t)

	data := gateway.DefaultIdentifyCommand("Bot " + env.BotToken)
	data.Shard = &gateway.Shard{0, env.ShardCount}
	data.Presence = &gateway.UpdatePresenceCommand{
		Status: discord.DoNotDisturbStatus,
		Activities: []discord.Activity{{
			Name: "Testing shards...",
			Type: discord.CustomActivity,
		}},
	}

	readyCh := make(chan *gateway.ReadyEvent)

	m, err := shard.NewIdentifiedManager(data, NewShardFunc(
		func(m *shard.Manager, s *State) {
			now := time.Now().Format(time.StampMilli)
			t.Log(now, "initializing shard")

			s.AddIntents(gateway.IntentGuilds)
			s.AddSyncHandler(readyCh)
			s.AddSyncHandler(func(err error) {
				t.Log("background error:", err)
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
