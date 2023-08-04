package defaultstore

import (
	"sync"

	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/state/store"
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

	g, ok := s.guilds[id]
	if ok {
		return &g, nil
	}

	return nil, store.ErrNotFound
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

func (s *Guild) GuildSet(guild *discord.Guild, update bool) error {
	cpy := *guild

	s.mut.Lock()
	if _, ok := s.guilds[guild.ID]; !ok || update {
		s.guilds[guild.ID] = cpy
	}
	s.mut.Unlock()

	return nil
}

func (s *Guild) GuildRemove(id discord.GuildID) error {
	s.mut.Lock()
	delete(s.guilds, id)
	s.mut.Unlock()
	return nil
}
