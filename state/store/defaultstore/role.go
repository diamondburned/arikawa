package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/diamondburned/arikawa/v3/state/store"
)

type Role struct {
	guilds moreatomic.Map
}

var _ store.RoleStore = (*Role)(nil)

type roles struct {
	mut   sync.Mutex
	roles map[discord.RoleID]discord.Role
}

func NewRole() *Role {
	return &Role{
		guilds: *moreatomic.NewMap(func() interface{} {
			return &roles{
				roles: make(map[discord.RoleID]discord.Role, 1),
			}
		}),
	}
}

func (s *Role) Reset() error {
	return s.guilds.Reset()
}

func (s *Role) Role(guildID discord.GuildID, roleID discord.RoleID) (*discord.Role, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	rs := iv.(*roles)

	rs.mut.Lock()
	defer rs.mut.Unlock()

	r, ok := rs.roles[roleID]
	if ok {
		return &r, nil
	}

	return nil, store.ErrNotFound
}

func (s *Role) Roles(guildID discord.GuildID) ([]discord.Role, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	rs := iv.(*roles)

	rs.mut.Lock()
	defer rs.mut.Unlock()

	var roles = make([]discord.Role, 0, len(rs.roles))
	for _, role := range rs.roles {
		roles = append(roles, role)
	}

	return roles, nil
}

func (s *Role) RoleSet(guildID discord.GuildID, role discord.Role) error {
	iv, _ := s.guilds.LoadOrStore(guildID)

	rs := iv.(*roles)

	rs.mut.Lock()
	rs.roles[role.ID] = role
	rs.mut.Unlock()

	return nil
}

func (s *Role) RoleRemove(guildID discord.GuildID, roleID discord.RoleID) error {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil
	}

	rs := iv.(*roles)

	rs.mut.Lock()
	delete(rs.roles, roleID)
	rs.mut.Unlock()

	return nil
}
