package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Me struct {
	self discord.User
	mut  sync.RWMutex
}

var _ store.MeStore = (*Me)(nil)

func NewMe() *Me {
	return &Me{}
}

func (m *Me) Reset() error {
	m.mut.Lock()
	m.self = discord.User{}
	m.mut.Unlock()

	return nil
}

func (m *Me) Me() (*discord.User, error) {
	m.mut.RLock()
	self := m.self
	m.mut.RUnlock()

	if !self.ID.IsValid() {
		return nil, store.ErrNotFound
	}

	return &self, nil
}

func (m *Me) MyselfSet(me discord.User) error {
	m.mut.Lock()
	m.self = me
	m.mut.Unlock()

	return nil
}
