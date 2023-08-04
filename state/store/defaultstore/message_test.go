package defaultstore

import (
	"testing"

	"libdb.so/arikawa/v4/discord"
)

func populate12Store() *Message {
	store := NewMessage(10)

	// Insert a regular list of messages.
	store.MessageSet(&discord.Message{ID: 1 << 29, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 28, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 27, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 26, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 25, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 24, ChannelID: 1}, false)

	// Try to insert newer messages after inserting new messages.
	store.MessageSet(&discord.Message{ID: 1 << 30, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 31, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 32, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 33, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 34, ChannelID: 1}, false)

	// TThese messages should be discarded, due to age.
	store.MessageSet(&discord.Message{ID: 1 << 23, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 22, ChannelID: 1}, false)

	// These should be prepended.
	store.MessageSet(&discord.Message{ID: 1 << 35, ChannelID: 1}, false)
	store.MessageSet(&discord.Message{ID: 1 << 36, ChannelID: 1}, false)

	return store
}

func TestMessageSet(t *testing.T) {
	store := populate12Store()

	messages, _ := store.Messages(1)
	if len(messages) < store.MaxMessages() {
		t.Errorf("store can store %d messages, but only returned %d", store.MaxMessages(),
			len(messages))
	}

	maxShift := 36

	for i, actual := range messages {
		expectID := discord.MessageID(1) << (maxShift - i)
		if actual.ID != expectID {
			t.Errorf("message at %d has mismatch ID %d, expecting %d", i, actual.ID, expectID)
		}
	}
}

func TestMessagesUpdate(t *testing.T) {
	store := populate12Store()

	store.MessageSet(&discord.Message{ID: 5, ChannelID: 1, Content: "edited 1"}, true)
	store.MessageSet(&discord.Message{ID: 6, ChannelID: 1, Content: "edited 2"}, true)
	store.MessageSet(&discord.Message{ID: 5, ChannelID: 1, Content: "edited 3"}, true)

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
