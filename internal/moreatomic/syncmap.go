package moreatomic

import (
	"sync"
	"sync/atomic"
)

// Map is a thread-safe map that is a wrapper around sync.Map with slight API
// additions.
type Map struct {
	smap atomic.Value
	ctor func() interface{}
}

type sentinelType struct{}

var sentinel = sentinelType{}

func NewMap(ctor func() interface{}) *Map {
	smap := atomic.Value{}
	smap.Store(&sync.Map{})
	return &Map{smap, ctor}
}

// Reset swaps the internal map out with a fresh one, dropping the old map. This
// method never errors.
func (sm *Map) Reset() error {
	sm.smap.Store(&sync.Map{})
	return nil
}

// LoadOrStore loads an existing value or stores a new value created from the
// given constructor then return that value.
func (sm *Map) LoadOrStore(k interface{}) (lv interface{}, loaded bool) {
	smap := sm.smap.Load().(*sync.Map)

	lv, loaded = smap.LoadOrStore(k, sentinel)
	if !loaded {
		lv = sm.ctor()
		smap.Store(k, lv)
	}

	return
}

// Load loads an existing value; it returns ok set to false if there is no
// value with that key.
func (sm *Map) Load(k interface{}) (lv interface{}, ok bool) {
	smap := sm.smap.Load().(*sync.Map)

	for {
		lv, ok = smap.Load(k)
		if !ok {
			return nil, false
		}

		if lv != sentinel {
			return lv, true
		}
	}
}
