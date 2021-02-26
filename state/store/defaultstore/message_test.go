package defaultstore

import (
	"testing"

	"github.com/diamondburned/arikawa/v2/discord"
)

func populate12Store() *Message {
	store := NewMessage(10)

	// Insert a regular list of messages.
	store.MessageSet(discord.Message{ID: 11, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 9, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 7, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 5, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 3, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 1, ChannelID: 1})

	// Try to insert newer messages after inserting new messages.
	store.MessageSet(discord.Message{ID: 12, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 10, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 8, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 6, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 4, ChannelID: 1})

	// These messages should be discarded.
	store.MessageSet(discord.Message{ID: 2, ChannelID: 1})
	store.MessageSet(discord.Message{ID: 0, ChannelID: 1})

	return store
}

func TestMessageSet(t *testing.T) {
	store := populate12Store()

	messages, _ := store.Messages(1)

	const (
		start discord.MessageID = 2
		end   discord.MessageID = 12
	)

	for i := start; i < end; i++ {
		index := i - start
		expect := end - i + start

		if msgID := messages[index].ID; msgID != expect {
			t.Errorf("message at %d has mismatch ID %d, expecting %d", i, msgID, expect)
		}
	}
}

func TestMessagesUpdate(t *testing.T) {
	store := populate12Store()

	store.MessageSet(discord.Message{ID: 5, ChannelID: 1, Content: "edited 1"})
	store.MessageSet(discord.Message{ID: 6, ChannelID: 1, Content: "edited 2"})
	store.MessageSet(discord.Message{ID: 5, ChannelID: 1, Content: "edited 3"})

	expect := map[discord.MessageID]string{
		5: "edited 3",
		6: "edited 2",
	}

	messages, _ := store.Messages(1)

	for i := 0; i < store.MaxMessages(); i++ {
		msg := messages[i]
		content, ok := expect[msg.ID]
		if ok && msg.Content != content {
			t.Errorf("id %d expected %q, got %q", i, content, msg.Content)
		}
	}
}
