package gateway

import "sync/atomic"

type Sequence int64

func NewSequence() *Sequence {
	return (*Sequence)(new(int64))
}

func (s *Sequence) Set(seq int64) { atomic.StoreInt64((*int64)(s), seq) }
func (s *Sequence) Get() int64    { return atomic.LoadInt64((*int64)(s)) }
