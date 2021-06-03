package moreatomic

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
)

type SnowflakeSet struct {
	set map[discord.Snowflake]struct{}
	mut sync.Mutex
}

// NewSnowflakeSet creates a new SnowflakeSet.
func NewSnowflakeSet() *SnowflakeSet {
	return &SnowflakeSet{
		set: make(map[discord.Snowflake]struct{}),
	}
}

// Add adds the passed discord.Snowflake to the set.
func (s *SnowflakeSet) Add(flake discord.Snowflake) {
	s.mut.Lock()

	s.set[flake] = struct{}{}

	s.mut.Unlock()
}

// Contains checks whether the passed discord.Snowflake is present in the set.
func (s *SnowflakeSet) Contains(flake discord.Snowflake) (ok bool) {
	s.mut.Lock()
	defer s.mut.Unlock()

	_, ok = s.set[flake]
	return
}

// Delete deletes the passed discord.Snowflake from the set and returns true if
// the element is present. If not, Delete is a no-op and returns false.
func (s *SnowflakeSet) Delete(flake discord.Snowflake) bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	if _, ok := s.set[flake]; ok {
		delete(s.set, flake)
		return true
	}

	return false
}
