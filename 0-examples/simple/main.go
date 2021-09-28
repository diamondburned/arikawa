// Package main demonstrates a bare simple bot without a state cache. It logs
// all messages it sees into stderr.
package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s := session.New("Bot " + token)
	s.AddHandler(func(c *gateway.MessageCreateEvent) {
		log.Println(c.Author.Username, "sent", c.Content)
	})

	// Add the needed Gateway intents.
	s.AddIntents(gateway.IntentGuildMessages)
	s.AddIntents(gateway.IntentDirectMessages)

	if err := s.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	u, err := s.Me()
	if err != nil {
		log.Fatalln("Failed to get myself:", err)
	}

	log.Println("Started as", u.Username)

	// Block forever.
	select {}
}
