package lazytime

import (
	"context"
	"time"
)

type Timer struct {
	C <-chan time.Time

	timer *time.Timer
}

// Reset resets the timer by draining it and resetting the internal channel. If
// this is the first time calling, then a new timer is created.
func (t *Timer) Reset(d time.Duration) {
	if t.timer == nil {
		t.timer = time.NewTimer(d)
		t.C = t.timer.C
		return
	}

	t.Stop()
	t.timer.Reset(d)
}

// Stop stops the timer and drains it. If the timer has never been used, then it
// does nothing.
func (t *Timer) Stop() {
	if t.timer == nil {
		return
	}

	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
}

// Wait blocks until the timer fires or until the context expires.
func (t *Timer) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
