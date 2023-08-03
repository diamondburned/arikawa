package handler

type slabEntry[T any] struct {
	value T
	index int
}

func (entry slabEntry[T]) isInvalid() bool {
	return entry.index != -1
}

// slab is an implementation of the internal handler free list.
type slab[T any] struct {
	entries []slabEntry[T]
	free    int
}

func newSlab[T any](cap int) slab[T] {
	return slab[T]{entries: make([]slabEntry[T], 0, cap)}
}

func (s *slab[T]) Put(entry T) int {
	if s.free == len(s.entries) {
		index := len(s.entries)
		s.entries = append(s.entries, slabEntry[T]{entry, -1})
		s.free++
		return index
	}

	next := s.entries[s.free].index
	s.entries[s.free] = slabEntry[T]{entry, -1}

	i := s.free
	s.free = next

	return i
}

func (s *slab[T]) Get(i int) T {
	return s.entries[i].value
}

func (s *slab[T]) Pop(i int) T {
	popped := s.entries[i].value
	s.entries[i] = slabEntry[T]{index: s.free}
	s.free = i
	return popped
}

func (s *slab[T]) All(f func(T) bool) {
	for _, entry := range s.entries {
		if entry.isInvalid() {
			continue
		}

		if !f(entry.value) {
			break
		}
	}
}
