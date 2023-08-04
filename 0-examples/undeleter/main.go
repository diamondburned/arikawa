// Package main demonstrates the PreHandler API of the State.
package main

import (
	"context"
	"log"
	"os"

	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/state"
	"libdb.so/arikawa/v4/utils/handler"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s := state.New("Bot " + token)
	// Make a pre-handler
	s.PreHandler = handler.New()
	s.PreHandler.AddSyncHandler(func(c *gateway.MessageDeleteEvent) {
		// Grab from the state
		m, err := s.Message(c.ChannelID, c.ID)
		if err != nil {
			log.Println("Not found:", c.ID)
		} else {
			log.Println(m.Author.Username, "deleted", m.Content)
		}
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
