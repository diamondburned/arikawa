package moreatomic

import (
	"context"

	"golang.org/x/sync/semaphore"
)

type BusyMutex struct {
	sema semaphore.Weighted
}

func NewBusyMutex() *BusyMutex {
	return &BusyMutex{
		sema: *semaphore.NewWeighted(1),
	}
}

func (m *BusyMutex) TryLock() bool {
	return m.sema.TryAcquire(1)
}

func (m *BusyMutex) IsBusy() bool {
	if !m.sema.TryAcquire(1) {
		return false
	}
	m.sema.Release(1)
	return true
}

func (m *BusyMutex) Lock(ctx context.Context) error {
	return m.sema.Acquire(ctx, 1)
}

func (m *BusyMutex) Unlock() {
	m.sema.Release(1)
}
