// Package backoff provides an exponential-backoff implementation partially
// taken from jpillora/backoff.
package backoff

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	factor = 2
	jitter = true
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Timer is a backoff timer.
type Timer struct {
	backoff Backoff
	timer   *time.Timer
}

// NewTimer returns a new uninitialized timer.
func NewTimer(min, max time.Duration) Timer {
	return Timer{
		backoff: NewBackoff(min, max),
	}
}

// Next initializes the timer if needed and returns a timer channel that fires
// when the backoff timeout is reached.
func (t *Timer) Next() <-chan time.Time {
	if t.timer == nil {
		t.timer = time.NewTimer(t.backoff.Next())
	} else {
		t.timer.Stop() // ensure drained
		t.timer.Reset(t.backoff.Next())
	}

	return t.timer.C
}

// Stop stops the internal timer and frees its resources. It does nothing if the
// timer is uninitialized.
func (t *Timer) Stop() {
	if t.timer == nil {
		return
	}

	if !t.timer.Stop() {
		<-t.timer.C // drain
	}
}

// Backoff is a time.Duration counter, starting at Min. After every call to
// the Duration method the current timing is multiplied by Factor, but it
// never exceeds Max.
type Backoff struct {
	min, max float64 // seconds
	attempt  int32   // negative == max uint32
}

// NewBackoff creates a new backoff time.Duration counter.
func NewBackoff(min, max time.Duration) Backoff {
	return Backoff{
		min: min.Seconds(),
		max: max.Seconds(),
	}
}

// Next returns the next backoff duration.
func (b *Backoff) Next() time.Duration {
	return b.forAttempt(atomic.AddInt32(&b.attempt, 1) - 1)
}

const maxInt64 = float64(math.MaxInt64 - 512)

// forAttempt returns the duration for a specific attempt. This is useful if
// you have a large number of independent Backoffs, but don't want use
// unnecessary memory storing the Backoff parameters per Backoff. The first
// attempt should be 0.
func (b *Backoff) forAttempt(attempt int32) time.Duration {
	if b.min >= b.max {
		// short-circuit
		return duration(b.max)
	}

	// Ensure attempt never overflows.
	if attempt < 0 {
		attempt = math.MaxInt32
	}

	// Calculate this duration.
	dur := b.min * math.Pow(factor, float64(attempt))
	if jitter {
		dur = rand.Float64()*(dur-b.min) + b.min
	}

	if dur < b.min {
		return duration(b.min)
	}
	if dur > b.max {
		return duration(b.max)
	}

	return duration(dur)
}

// duration converts a seconds float64 to time.Duration without losing accuracy.
func duration(secs float64) time.Duration {
	int, frac := math.Modf(secs)
	return (time.Duration(int) * time.Second) + time.Duration(frac*float64(time.Second))
}
