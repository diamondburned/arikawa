package gateway

import "sync/atomic"

type Sequence struct {
	seq int64
}

func NewSequence() *Sequence {
	return &Sequence{0}
}

func (s *Sequence) Set(seq int64) { atomic.StoreInt64(&s.seq, seq) }
func (s *Sequence) Get() int64    { return atomic.LoadInt64(&s.seq) }
