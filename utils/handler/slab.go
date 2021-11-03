package handler

type slabEntry struct {
	index int
	handler
}

func (entry slabEntry) isInvalid() bool {
	return entry.index != -1
}

// slab is an implementation of the internal handler free list.
type slab struct {
	Entries []slabEntry
	free    int
}

func (s *slab) Put(entry handler) int {
	if s.free == len(s.Entries) {
		index := len(s.Entries)
		s.Entries = append(s.Entries, slabEntry{-1, entry})
		s.free++
		return index
	}

	next := s.Entries[s.free].index
	s.Entries[s.free] = slabEntry{-1, entry}

	i := s.free
	s.free = next

	return i
}

func (s *slab) Get(i int) handler {
	return s.Entries[i].handler
}

func (s *slab) Pop(i int) handler {
	popped := s.Entries[i].handler
	s.Entries[i] = slabEntry{s.free, handler{}}
	s.free = i
	return popped
}
