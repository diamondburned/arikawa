package moreatomic

import "sync/atomic"

type Int64 int64

func NewInt64(v int64) *Int64 {
	i := new(Int64)
	*i = Int64(v)
	return i
}

func (i *Int64) Set(v int64) { atomic.StoreInt64((*int64)(i), v) }
func (i *Int64) Get() int64  { return atomic.LoadInt64((*int64)(i)) }
