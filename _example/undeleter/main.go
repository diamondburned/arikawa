package main

import (
	"log"
	"os"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
	"github.com/diamondburned/arikawa/state"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s, err := state.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	// Make a pre-handler
	s.PreHandler = handler.New()
	s.PreHandler.Synchronous = true
	s.PreHandler.AddHandler(func(c *gateway.MessageDeleteEvent) {
		// Grab from the state
		m, err := s.Message(c.ChannelID, c.ID)
		if err != nil {
			log.Println("Not found:", c.ID)
		} else {
			log.Println(m.Author.Username, "deleted", m.Content)
		}
	})

	if err := s.Open(); err != nil {
		log.Fatalln("Failed to connect:", err)
	}

	u, err := s.Me()
	if err != nil {
		log.Fatalln("Failed to get myself:", err)
	}

	log.Println("Started as", u.Username)

	// Block until a fatal error or SIGINT. Wait also calls Close().
	if err := s.Wait(); err != nil {
		log.Fatalln("Gateway fatal error:", err)
	}
}
