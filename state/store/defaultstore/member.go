package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Member struct {
	guilds moreatomic.Map // discord.GuildID -> *guildMembers
}

type guildMembers struct {
	mut     sync.Mutex
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

	gm.mut.Lock()
	defer gm.mut.Unlock()

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

	gm.mut.Lock()
	defer gm.mut.Unlock()

	var members = make([]discord.Member, 0, len(gm.members))
	for _, m := range gm.members {
		members = append(members, m)
	}

	return members, nil
}

func (s *Member) MemberSet(guildID discord.GuildID, member discord.Member) error {
	iv, _ := s.guilds.LoadOrStore(guildID)
	gm := iv.(*guildMembers)

	gm.mut.Lock()
	gm.members[member.User.ID] = member
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
