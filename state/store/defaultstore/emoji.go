package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Emoji struct {
	guilds moreatomic.Map
}

type emojis struct {
	mut    sync.Mutex
	emojis []discord.Emoji
}

var _ store.EmojiStore = (*Emoji)(nil)

func NewEmoji() *Emoji {
	return &Emoji{
		guilds: *moreatomic.NewMap(func() interface{} {
			return &emojis{
				emojis: []discord.Emoji{},
			}
		}),
	}
}

func (s *Emoji) Reset() error {
	s.guilds.Reset()
	return nil
}

func (s *Emoji) Emoji(guildID discord.GuildID, emojiID discord.EmojiID) (*discord.Emoji, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	es := iv.(*emojis)

	es.mut.Lock()
	defer es.mut.Unlock()

	for _, emoji := range es.emojis {
		if emoji.ID == emojiID {
			// Emoji is an implicit copy made by range, so we could do this
			// safely.
			return &emoji, nil
		}
	}

	return nil, store.ErrNotFound
}

func (s *Emoji) Emojis(guildID discord.GuildID) ([]discord.Emoji, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	es := iv.(*emojis)

	es.mut.Lock()
	defer es.mut.Unlock()

	// We're never modifying the slice internals ourselves, so this is fine.
	return es.emojis, nil
}

func (s *Emoji) EmojiSet(guildID discord.GuildID, allEmojis []discord.Emoji) error {
	iv, _ := s.guilds.LoadOrStore(guildID)

	es := iv.(*emojis)

	es.mut.Lock()
	es.emojis = allEmojis
	es.mut.Unlock()

	return nil
}
