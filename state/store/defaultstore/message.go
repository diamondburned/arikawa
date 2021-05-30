package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Message struct {
	channels moreatomic.Map
	maxMsgs  int
}

var _ store.MessageStore = (*Message)(nil)

type messages struct {
	mut      sync.Mutex
	messages []discord.Message
}

func NewMessage(maxMsgs int) *Message {
	return &Message{
		channels: *moreatomic.NewMap(func() interface{} {
			return &messages{
				messages: []discord.Message{}, // never use a nil slice
			}
		}),
		maxMsgs: maxMsgs,
	}
}

func (s *Message) Reset() error {
	return s.channels.Reset()
}

func (s *Message) Message(chID discord.ChannelID, mID discord.MessageID) (*discord.Message, error) {
	iv, ok := s.channels.Load(chID)
	if !ok {
		return nil, store.ErrNotFound
	}

	msgs := iv.(*messages)

	msgs.mut.Lock()
	defer msgs.mut.Unlock()

	for _, m := range msgs.messages {
		if m.ID == mID {
			return &m, nil
		}
	}

	return nil, store.ErrNotFound
}

func (s *Message) Messages(channelID discord.ChannelID) ([]discord.Message, error) {
	iv, ok := s.channels.Load(channelID)
	if !ok {
		return nil, store.ErrNotFound
	}

	msgs := iv.(*messages)

	msgs.mut.Lock()
	defer msgs.mut.Unlock()

	return append([]discord.Message(nil), msgs.messages...), nil
}

func (s *Message) MaxMessages() int {
	return s.maxMsgs
}

func (s *Message) MessageSet(message discord.Message, update bool) error {
	if s.maxMsgs <= 0 {
		return nil
	}

	iv, _ := s.channels.LoadOrStore(message.ChannelID)

	msgs := iv.(*messages)

	msgs.mut.Lock()
	defer msgs.mut.Unlock()

	if update {
		// Opt for a linear latest-to-oldest search in favor of something like
		// sort.Search, since more recent messages are more likely to be edited
		// than older ones.
		for i, oldMessage := range msgs.messages {
			// We found a match, update it.
			if oldMessage.ID == message.ID {
				DiffMessage(message, &oldMessage)
				msgs.messages[i] = oldMessage // Now updated.
				return nil
			}
		}

		return nil
	}

	switch {
	case len(msgs.messages) == 0:
		msgs.messages = []discord.Message{message}
	case shouldPrependMessage(message, msgs.messages):
		if len(msgs.messages) == s.maxMsgs {
			copy(msgs.messages[1:], msgs.messages)
			msgs.messages[0] = message
		} else {
			msgs.messages = append([]discord.Message{message}, msgs.messages...)
		}
	case shouldAppendMessage(message, msgs.messages) && len(msgs.messages) < s.maxMsgs:
		msgs.messages = append(msgs.messages, message)
	}

	// We already have this message or we can't append any more messages.
	return nil
}

// shouldPrependMessage checks if the passed message may be prepended to the
// passed message slice, that is ordered from latest to oldest.
//
// shouldPrependMessage is biased as it will return true if the timestamps
// of the passed message and the latest message match, even though the true
// order cannot be determined in that case.
func shouldPrependMessage(message discord.Message, messages []discord.Message) bool {
	// The id of message is greater than the one of the first aka. newest
	// message. It is therefore younger, and should be inserted.
	if message.ID > messages[0].ID {
		return true
	}

	// Two cases remain, the timestamps are equal or message is older than our
	// first message. So we compare timestamps. If they are equal, make sure
	// messages doesn't contain a message with the same id, in order to prevent
	// insertion of a duplicate.
	// ID timestamps are used, as they provide millisecond accuracy in contrast
	// to the second accuracy of discord.Message.Timestamp.
	ts := message.ID >> 22

	if ts == messages[0].ID>>22 {
		// Only iterate as long as timestamps are equal, or there are no more
		// messages.
		for i := 0; i < len(messages) && messages[i].ID>>22 == ts; i++ {
			// Duplicate, don't insert.
			if messages[i].ID == message.ID {
				return false
			}
		}

		// No duplicate of message found, so safe to prepend.
		return true
	}

	// Message is older than our most recent message, don't prepend.
	return false
}

// shouldAppendMessage checks if the passed message may be appended to the
// passed message slice, that is ordered from latest to oldest.
//
// shouldPrependMessage is biased as it will return true if the timestamps
// of the passed message and the oldest message match, even though the true
// order cannot be determined in that case.
func shouldAppendMessage(message discord.Message, messages []discord.Message) bool {
	// The id of message is smaller than the one of the last aka. oldest
	// message. It is therefore older, and should be inserted.
	if message.ID < messages[len(messages)-1].ID {
		return true
	}

	// Two cases remain, the timestamps are equal or message is younger than
	// our last message. So we compare timestamps. If they are equal, make sure
	// messages doesn't contain a message with the same id, in order to prevent
	// insertion of a duplicate.
	// ID timestamps are used, as they provide millisecond accuracy in contrast
	// to the second accuracy of discord.Message.Timestamp.
	ts := message.ID << 22

	// Timestamps are equal, check for duplicates.
	if ts == messages[len(messages)-1].ID>>22 {
		// Only iterate as long as timestamps are equal, or there are no more
		// messages.
		for i := len(messages) - 1; i >= 0 && messages[i].ID>>22 == ts; i-- {
			// Duplicate, don't insert.
			if messages[i].ID == message.ID {
				return false
			}
		}

		// No duplicate of message found, so safe to append.
		return true
	}

	// Message is younger than our oldest message, don't append.
	return false
}

// DiffMessage fills non-empty fields from src to dst.
func DiffMessage(src discord.Message, dst *discord.Message) {
	// Thanks, Discord.
	if src.Content != "" {
		dst.Content = src.Content
	}
	if src.EditedTimestamp.IsValid() {
		dst.EditedTimestamp = src.EditedTimestamp
	}
	if src.Mentions != nil {
		dst.Mentions = src.Mentions
	}
	if src.Embeds != nil {
		dst.Embeds = src.Embeds
	}
	if src.Attachments != nil {
		dst.Attachments = src.Attachments
	}
	if src.Timestamp.IsValid() {
		dst.Timestamp = src.Timestamp
	}
	if src.Author.ID.IsValid() {
		dst.Author = src.Author
	}
	if src.Reactions != nil {
		dst.Reactions = src.Reactions
	}
	if src.Components != nil {
		dst.Components = src.Components
	}
}

func (s *Message) MessageRemove(channelID discord.ChannelID, messageID discord.MessageID) error {
	iv, ok := s.channels.Load(channelID)
	if !ok {
		return nil
	}

	msgs := iv.(*messages)

	msgs.mut.Lock()
	defer msgs.mut.Unlock()

	for i, m := range msgs.messages {
		if m.ID == messageID {
			msgs.messages = append(msgs.messages[:i], msgs.messages[i+1:]...)
			return nil
		}
	}

	return nil
}
