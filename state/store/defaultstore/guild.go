package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state/store"
)

type Guild struct {
	mut    sync.RWMutex
	guilds map[discord.GuildID]discord.Guild
}

var _ store.GuildStore = (*Guild)(nil)

func NewGuild() *Guild {
	return &Guild{
		guilds: map[discord.GuildID]discord.Guild{},
	}
}

func (s *Guild) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.guilds = map[discord.GuildID]discord.Guild{}

	return nil
}

func (s *Guild) Guild(id discord.GuildID) (*discord.Guild, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ch, ok := s.guilds[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	// implicit copy
	return &ch, nil
}

func (s *Guild) Guilds() ([]discord.Guild, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if len(s.guilds) == 0 {
		return nil, store.ErrNotFound
	}

	var gs = make([]discord.Guild, 0, len(s.guilds))
	for _, g := range s.guilds {
		gs = append(gs, g)
	}

	return gs, nil
}

func (s *Guild) GuildSet(guild discord.Guild) error {
	s.mut.Lock()
	s.guilds[guild.ID] = guild
	s.mut.Unlock()
	return nil
}

func (s *Guild) GuildRemove(id discord.GuildID) error {
	s.mut.Lock()
	delete(s.guilds, id)
	s.mut.Unlock()
	return nil
}
