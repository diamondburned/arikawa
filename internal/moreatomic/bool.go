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

func (b *Bool) CompareAndSwap(old bool) bool {
	var oldN uint32 = 0
	if old {
		oldN = 1
	}

	return atomic.CompareAndSwapUint32(&b.val, oldN, (oldN+1)%2)
}
