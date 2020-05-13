// +build integration

package api

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/discord"
)

type testConfig struct {
	BotToken  string
	ChannelID discord.Snowflake
}

func mustConfig(t *testing.T) testConfig {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	var cid = os.Getenv("CHANNEL_ID")
	if cid == "" {
		t.Fatal("Missing $CHANNEL_ID")
	}

	id, err := discord.ParseSnowflake(cid)
	if err != nil {
		t.Fatal("Invalid $CHANNEL_ID:", err)
	}

	return testConfig{
		BotToken:  token,
		ChannelID: id,
	}
}

func TestIntegration(t *testing.T) {
	cfg := mustConfig(t)

	client := NewClient("Bot " + cfg.BotToken)

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

var emojisToSend = [...]string{
	"🥺",
	"❤",
	"😂",
	"🥰",
	"😊",
	"🔥",
	"✔",
	"👍",
	"😍",
	"🐻",
	"🤯",
	"🔣",
	"🍔",
	"🎌",
	"🇯🇵",
	"🎥",
	"🇺🇸",
	"🌎",
}

func TestReactions(t *testing.T) {
	cfg := mustConfig(t)

	client := NewClient("Bot " + cfg.BotToken)

	msg := fmt.Sprintf("This is a message sent at %v.", time.Now())

	// Send a new message.
	m, err := client.SendMessage(cfg.ChannelID, msg, nil)
	if err != nil {
		t.Fatal("Failed to send message:", err)
	}

	now := time.Now()

	for _, emojiString := range emojisToSend {
		if err := client.React(cfg.ChannelID, m.ID, emojiString); err != nil {
			t.Fatal("Failed to send emoji "+emojiString+":", err)
		}
	}

	msg += fmt.Sprintf(" Total time taken to send all reactions: %v.", time.Now().Sub(now))

	m, err = client.EditMessage(cfg.ChannelID, m.ID, msg, nil, false)
	if err != nil {
		t.Fatal("Failed to edit message:", err)
	}
}
