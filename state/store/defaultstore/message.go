package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/diamondburned/arikawa/v3/state/store"
)

type Message struct {
	channels moreatomic.Map
	maxMsgs  int
}

var _ store.MessageStore = (*Message)(nil)

type messages struct {
	mut      sync.RWMutex
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

	msgs.mut.RLock()
	defer msgs.mut.RUnlock()

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

	msgs.mut.RLock()
	defer msgs.mut.RUnlock()

	return append([]discord.Message(nil), msgs.messages...), nil
}

func (s *Message) MaxMessages() int {
	return s.maxMsgs
}

func (s *Message) MessageSet(message *discord.Message, update bool) error {
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

	if len(msgs.messages) == 0 {
		msgs.messages = []discord.Message{*message}
	}

	if pos := messageInsertPosition(message, msgs.messages); pos < 0 {
		// Messages are full, drop the oldest messages to make room.
		if len(msgs.messages) == s.maxMsgs {
			copy(msgs.messages[1:], msgs.messages)
			msgs.messages[0] = *message
		} else {
			msgs.messages = append([]discord.Message{*message}, msgs.messages...)
		}
	} else if pos > 0 && len(msgs.messages) < s.maxMsgs {
		msgs.messages = append(msgs.messages, *message)
	}

	// We already have this message or we can't append any more messages.
	return nil
}

// messageInsertPosition checks if the message should be appended or prepended
// into the passed messages, ordered by time of creation from latest to oldest.
// If the message should be prepended, messageInsertPosition returns -1, and if
// the message should be appended it returns 1. As a third option it returns 0,
// if the message should not be added to the slice, because it would disrupt
// the order.
//
// messageInsertPosition is biased as it will recommend adding the message even
// if timestamps just match, even though the true order cannot be determined in
// that case.
func messageInsertPosition(target *discord.Message, messages []discord.Message) int8 {
	var (
		targetTime = target.ID.Time()
		firstTime  = messages[0].ID.Time()
		lastTime   = messages[len(messages)-1].ID.Time()
	)

	if targetTime.After(firstTime) {
		return -1
	} else if targetTime.Before(lastTime) {
		return 1
	}

	// Two cases remain, the timestamp is equal to either the latest or oldest
	// message, or the message is already contained in message.
	// So we compare timestamps. If they are equal, make sure messages doesn't
	// contain a message with the same id, in order to prevent insertion of a
	// duplicate. If they are not equal, we return 0 as the message would
	// violate the order of messages.
	// ID timestamps are used, as they provide millisecond accuracy in contrast
	// to the second accuracy of discord.Message.Timestamp.
	if targetTime.Equal(firstTime) {
		// Only iterate as long as timestamps are equal, or there are no more
		// messages.
		for i := 0; i < len(messages) && targetTime.Equal(messages[i].ID.Time()); i++ {
			// Duplicate, don't insert.
			if messages[i].ID == target.ID {
				return 0
			}
		}

		// No duplicate of message found, so safe to prepend.
		return -1
	} else if targetTime.Equal(lastTime) {
		// Only iterate as long as timestamps are equal, or there are no more
		// messages.
		for i := len(messages) - 1; i >= 0 && targetTime.Equal(messages[i].ID.Time()); i-- {
			// Duplicate, don't insert.
			if messages[i].ID == target.ID {
				return 0
			}
		}

		// No duplicate of message found, so safe to append.
		return 1
	}

	// Message would violate the order of messages, don't add it.
	return 0
}

// DiffMessage fills non-empty fields from src to dst.
func DiffMessage(src, dst *discord.Message) {
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
