package moreatomic

import (
	"context"
)

type CtxMutex struct {
	mut chan struct{}
}

func NewCtxMutex() *CtxMutex {
	return &CtxMutex{
		mut: make(chan struct{}, 1),
	}
}

// func (m *CtxMutex) TryLock() bool {
// 	select {
// 	case m.mut <- struct{}{}:
// 		return true
// 	default:
// 		return false
// 	}
// }

// func (m *CtxMutex) IsBusy() bool {
// 	select {
// 	case m.mut <- struct{}{}:
// 		<-m.mut
// 		return false
// 	default:
// 		return true
// 	}
// }

func (m *CtxMutex) Lock(ctx context.Context) error {
	select {
	case m.mut <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *CtxMutex) Unlock() {
	select {
	case <-m.mut:
		// return
	default:
		panic("Unlock of already unlocked mutex.")
	}
}
