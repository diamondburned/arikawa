package defaultstore

import (
	"sync"

	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/internal/moreatomic"
	"libdb.so/arikawa/v4/state/store"
)

type Member struct {
	guilds moreatomic.Map // discord.GuildID -> *guildMembers
}

type guildMembers struct {
	mut     sync.RWMutex
	members map[discord.UserID]discord.Member
}

var _ store.MemberStore = (*Member)(nil)

func NewMember() *Member {
	return &Member{
		guilds: *moreatomic.NewMap(func() interface{} {
			return &guildMembers{
				members: make(map[discord.UserID]discord.Member, 1),
			}
		}),
	}
}

func (s *Member) Reset() error {
	return s.guilds.Reset()
}

func (s *Member) Member(guildID discord.GuildID, userID discord.UserID) (*discord.Member, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	gm := iv.(*guildMembers)

	gm.mut.RLock()
	defer gm.mut.RUnlock()

	m, ok := gm.members[userID]
	if ok {
		return &m, nil
	}

	return nil, store.ErrNotFound
}

func (s *Member) Members(guildID discord.GuildID) ([]discord.Member, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	gm := iv.(*guildMembers)

	gm.mut.RLock()
	defer gm.mut.RUnlock()

	var members = make([]discord.Member, 0, len(gm.members))
	for _, m := range gm.members {
		members = append(members, m)
	}

	return members, nil
}

func (s *Member) MemberSet(guildID discord.GuildID, m *discord.Member, update bool) error {
	iv, _ := s.guilds.LoadOrStore(guildID)
	gm := iv.(*guildMembers)

	gm.mut.Lock()
	if _, ok := gm.members[m.User.ID]; !ok || update {
		gm.members[m.User.ID] = *m
	}
	gm.mut.Unlock()

	return nil
}

func (s *Member) MemberRemove(guildID discord.GuildID, userID discord.UserID) error {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil
	}

	gm := iv.(*guildMembers)

	gm.mut.Lock()
	delete(gm.members, userID)
	gm.mut.Unlock()

	return nil
}
