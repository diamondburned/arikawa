package state_test

import (
	"context"
	"log"
	"os"
	"os/signal"

	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/state"
)

func Example() {
	s := state.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	s.AddIntents(gateway.IntentGuilds | gateway.IntentGuildMessages)
	s.AddHandler(func(m *gateway.MessageCreateEvent) {
		log.Printf("%s: %s", m.Author.Username, m.Content)
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := s.Open(ctx); err != nil {
		log.Println("cannot open:", err)
	}

	<-ctx.Done() // block until Ctrl+C

	if err := s.Close(); err != nil {
		log.Println("cannot close:", err)
	}
}
