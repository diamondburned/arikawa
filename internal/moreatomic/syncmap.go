package moreatomic

import "github.com/kawasin73/umutex"

// Map is a thread-safe map that is a wrapper around sync.Map with slight API
// additions.
type Map struct {
	upmu umutex.UMutex
	smap map[interface{}]interface{}
	ctor func() interface{}
}

type sentinelType struct{}

var sentinel = sentinelType{}

func NewMap(ctor func() interface{}) *Map {
	return &Map{
		smap: map[interface{}]interface{}{},
		ctor: ctor,
	}
}

// Reset swaps the internal map out with a fresh one, dropping the old map. This
// method never errors.
func (sm *Map) Reset() error {
	sm.upmu.Lock()
	sm.smap = map[interface{}]interface{}{}
	sm.upmu.Unlock()
	return nil
}

// LoadOrStore loads an existing value or stores a new value created from the
// given constructor then return that value.
func (sm *Map) LoadOrStore(k interface{}) (lv interface{}, loaded bool) {
	sm.upmu.RLock()

	lv, loaded = sm.smap[k]
	if !loaded {
		lv = sm.ctor()

		// Wait until upgrade succeeds.
		for !sm.upmu.Upgrade() {
		}

		sm.smap[k] = lv

		sm.upmu.Unlock()
		return
	}

	sm.upmu.RUnlock()
	return
}

// Load loads an existing value; it returns ok set to false if there is no
// value with that key.
func (sm *Map) Load(k interface{}) (lv interface{}, ok bool) {
	sm.upmu.RLock()
	defer sm.upmu.RUnlock()

	lv, ok = sm.smap[k]
	return
}
