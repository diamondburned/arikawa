package api

import (
	"fmt"
	"log"
	"testing"
	"time"

	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/internal/testenv"
)

func TestIntegration(t *testing.T) {
	cfg := testenv.Must(t)

	client := NewClient("Bot " + cfg.BotToken)

	// Simple GET request
	u, err := client.Me()
	if err != nil {
		t.Fatal("Can't get self:", err)
	}

	log.Println("API user:", u.Username)

	// POST with URL param and paginator
	guilds, err := client.Guilds(100)
	if err != nil {
		t.Fatal("Can't get guilds:", err)
	}

	for _, guild := range guilds {
		if !guild.ID.IsValid() {
			t.Errorf("guild %q has invalid ID", guild.Name)
			continue
		}

		channels, err := client.Channels(guild.ID)
		if err != nil {
			t.Errorf(
				"failed to fetch channels for guild %q (%v): %v",
				guild.Name, guild.ID, err,
			)
		}

		for _, ch := range channels {
			if !ch.ID.IsValid() {
				t.Errorf(
					"channel %q of guild %q (%v) has invalid ID",
					ch.Name, guild.Name, guild.ID,
				)
			}
		}
	}
}

var emojisToSend = [...]discord.APIEmoji{
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
	cfg := testenv.Must(t)

	client := NewClient("Bot " + cfg.BotToken)

	msg := fmt.Sprintf("This is a message sent at %v.", time.Now())

	// Send a new message.
	m, err := client.SendMessage(cfg.ChannelID, msg)
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

	m, err = client.EditMessage(cfg.ChannelID, m.ID, msg)
	if err != nil {
		t.Fatal("Failed to edit message:", err)
	}
}
