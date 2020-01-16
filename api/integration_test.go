// +build integration

package api

import (
	"log"
	"os"
	"testing"
)

func TestIntegration(t *testing.T) {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	client := NewClient("Bot " + token)

	// Simple GET request
	u, err := client.Me()
	if err != nil {
		t.Fatal("Can't get self:", err)
	}

	log.Println("API user:", u.Username)

	// POST with URL param and paginator
	_, err = client.Guilds(100)
	if err != nil {
		t.Fatal("Can't get guilds:", err)
	}
}
