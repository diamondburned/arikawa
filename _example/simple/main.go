// Package main demonstrates a bare simple bot without a state cache. It logs
// all messages it sees into stderr.
package main

import (
	"log"
	"os"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s, err := session.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	s.AddHandler(func(c *gateway.MessageCreateEvent) {
		log.Println(c.Author.Username, "sent", c.Content)
	})

	if err := s.Open(); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	// Add the needed Gateway intents.
	s.Gateway.AddIntents(gateway.IntentGuildMessages)
	s.Gateway.AddIntents(gateway.IntentDirectMessages)

	u, err := s.Me()
	if err != nil {
		log.Fatalln("Failed to get myself:", err)
	}

	log.Println("Started as", u.Username)

	// Block forever.
	select {}
}
