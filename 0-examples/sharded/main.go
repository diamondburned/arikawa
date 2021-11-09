// Package main demonstrates a bare simple bot without a state cache. It logs
// all messages it sees into stderr.
package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/state"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	newShard := state.NewShardFunc(func(m *shard.Manager, s *state.State) {
		// Add the needed Gateway intents.
		s.AddIntents(gateway.IntentGuildMessages)
		s.AddIntents(gateway.IntentDirectMessages)

		s.AddHandler(func(c *gateway.MessageCreateEvent) {
			_, shardIx := m.FromGuildID(c.GuildID)
			log.Println(c.Author.Tag(), "sent", c.Content, "on shard", shardIx)
		})
	})

	m, err := shard.NewManager("Bot "+token, newShard)
	if err != nil {
		log.Fatalln("failed to create shard manager:", err)
	}

	if err := m.Open(context.Background()); err != nil {
		log.Fatalln("failed to connect shards:", err)
	}
	defer m.Close()

	var shardNum int

	m.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		u, err := state.Me()
		if err != nil {
			log.Fatalln("failed to get myself:", err)
		}

		log.Printf("Shard %d/%d started as %s", shardNum, m.NumShards()-1, u.Tag())

		shardNum++
	})

	// Block forever.
	select {}
}
