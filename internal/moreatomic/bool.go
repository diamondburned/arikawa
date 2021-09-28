package moreatomic

import "sync/atomic"

type Bool struct {
	val uint32
}

func (b *Bool) Get() bool {
	return atomic.LoadUint32(&b.val) > 0
}

func (b *Bool) Set(val bool) {
	var x = uint32(0)
	if val {
		x = 1
	}
	atomic.StoreUint32(&b.val, x)
}

func (b *Bool) SetTrue() {
	atomic.StoreUint32(&b.val, 1)
}

func (b *Bool) SetFalse() {
	atomic.StoreUint32(&b.val, 0)
}

// Acquire sets bool to true if it's false and returns true, otherwise returns
// false.
func (b *Bool) Acquire() bool {
	return atomic.CompareAndSwapUint32(&b.val, 0, 1)
}
