package moreatomic

import (
	"sync/atomic"

	"github.com/diamondburned/arikawa/v3/internal/moreatomic/syncmod"
)

// Map is a thread-safe map that is a wrapper around sync.Map with slight API
// additions.
type Map struct {
	val  atomic.Value
	ctor func() any
}

func NewMap(ctor func() any) *Map {
	sm := &Map{ctor: ctor}
	sm.Reset()
	return sm
}

// Reset swaps the internal map out with a fresh one, dropping the old map. This
// method never errors.
func (sm *Map) Reset() error {
	sm.val.Store(&syncmod.Map{New: sm.ctor})
	return nil
}

// LoadOrStore loads an existing value or stores a new value created from the
// given constructor then return that value.
func (sm *Map) LoadOrStore(k any) (lv any, loaded bool) {
	return sm.val.Load().(*syncmod.Map).LoadOrStore(k)
}

// Load loads an existing value; it returns ok set to false if there is no
// value with that key.
func (sm *Map) Load(k any) (lv any, ok bool) {
	return sm.val.Load().(*syncmod.Map).Load(k)
}
