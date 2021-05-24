package moreatomic

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
)

type GuildIDSet struct {
	set map[discord.GuildID]struct{}
	mut sync.Mutex
}

// NewGuildIDSet creates a new GuildIDSet.
func NewGuildIDSet() *GuildIDSet {
	return &GuildIDSet{
		set: make(map[discord.GuildID]struct{}),
	}
}

// Add adds the passed discord.GuildID to the set.
func (s *GuildIDSet) Add(flake discord.GuildID) {
	s.mut.Lock()

	s.set[flake] = struct{}{}

	s.mut.Unlock()
}

// Contains checks whether the passed discord.GuildID is present in the set.
func (s *GuildIDSet) Contains(flake discord.GuildID) (ok bool) {
	s.mut.Lock()
	defer s.mut.Unlock()

	_, ok = s.set[flake]
	return
}

// Delete deletes the passed discord.GuildID from the set and returns true if
// the element is present. If not, Delete is a no-op and returns false.
func (s *GuildIDSet) Delete(flake discord.GuildID) bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	if _, ok := s.set[flake]; ok {
		delete(s.set, flake)
		return true
	}

	return false
}

// Clear deletes all elements from the set.
func (s *GuildIDSet) Clear() {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.set = make(map[discord.GuildID]struct{})
}
