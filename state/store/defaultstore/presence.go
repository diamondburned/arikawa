package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Presence struct {
	guilds moreatomic.Map
}

type presences struct {
	presences map[discord.UserID]gateway.Presence
	mut       sync.Mutex
}

var _ store.PresenceStore = (*Presence)(nil)

func NewPresence() *Presence {
	return &Presence{
		guilds: *moreatomic.NewMap(func() interface{} {
			return &presences{
				presences: make(map[discord.UserID]gateway.Presence, 1),
			}
		}),
	}
}

func (s *Presence) Reset() error {
	return s.guilds.Reset()
}

func (s *Presence) Presence(gID discord.GuildID, uID discord.UserID) (*gateway.Presence, error) {
	iv, ok := s.guilds.Load(gID)
	if !ok {
		return nil, store.ErrNotFound
	}

	ps := iv.(*presences)

	ps.mut.Lock()
	defer ps.mut.Unlock()

	p, ok := ps.presences[uID]
	if ok {
		return &p, nil
	}

	return nil, store.ErrNotFound
}

func (s *Presence) Presences(guildID discord.GuildID) ([]gateway.Presence, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	ps := iv.(*presences)

	ps.mut.Lock()
	defer ps.mut.Unlock()

	var presences = make([]gateway.Presence, 0, len(ps.presences))
	for _, p := range ps.presences {
		presences = append(presences, p)
	}

	return presences, nil
}

func (s *Presence) PresenceSet(guildID discord.GuildID, presence gateway.Presence) error {
	iv, _ := s.guilds.LoadOrStore(guildID)

	ps := iv.(*presences)

	ps.mut.Lock()
	defer ps.mut.Unlock()

	// Shitty if check is better than a realloc every time.
	if ps.presences == nil {
		ps.presences = make(map[discord.UserID]gateway.Presence, 1)
	}

	ps.presences[presence.User.ID] = presence

	return nil
}

func (s *Presence) PresenceRemove(guildID discord.GuildID, userID discord.UserID) error {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil
	}

	ps := iv.(*presences)

	ps.mut.Lock()
	delete(ps.presences, userID)
	ps.mut.Unlock()

	return nil
}
