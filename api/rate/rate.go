package rate

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/utils/moreatomic"
	"github.com/pkg/errors"
)

// ExtraDelay because Discord is trash. I've seen this in both litcord and
// discordgo, with dgo claiming from his experiments.
// RE: Those who want others to fix it for them: release the source code then.
const ExtraDelay = 250 * time.Millisecond

// This makes me suicidal.
// https://github.com/bwmarrin/discordgo/blob/master/ratelimit.go

type Limiter struct {
	// Only 1 per bucket
	CustomLimits []*CustomRateLimit

	Prefix string

	global     *int64 // atomic guarded, unixnano
	buckets    sync.Map
	globalRate time.Duration
}

type CustomRateLimit struct {
	Contains string
	Reset    time.Duration
}

type bucket struct {
	lock   moreatomic.CtxMutex
	custom *CustomRateLimit

	remaining uint64
	limit     uint

	reset     time.Time
	lastReset time.Time // only for custom
}

func newBucket() *bucket {
	return &bucket{
		lock:      *moreatomic.NewCtxMutex(),
		remaining: 1,
	}
}

func NewLimiter(prefix string) *Limiter {
	return &Limiter{
		Prefix:       prefix,
		global:       new(int64),
		buckets:      sync.Map{},
		CustomLimits: []*CustomRateLimit{},
	}
}

func (l *Limiter) getBucket(path string, store bool) *bucket {
	path = ParseBucketKey(strings.TrimPrefix(path, l.Prefix))

	bc, ok := l.buckets.Load(path)
	if !ok && !store {
		return nil
	}

	if !ok {
		bc := newBucket()

		for _, limit := range l.CustomLimits {
			if strings.Contains(path, limit.Contains) {
				bc.custom = limit
				break
			}
		}

		l.buckets.Store(path, bc)
		return bc
	}

	return bc.(*bucket)
}

func (l *Limiter) Acquire(ctx context.Context, path string) error {
	b := l.getBucket(path, true)

	if err := b.lock.Lock(ctx); err != nil {
		return err
	}

	// Time to sleep
	var sleep time.Duration

	if b.remaining == 0 && b.reset.After(time.Now()) {
		// out of turns, gotta wait
		sleep = b.reset.Sub(time.Now())
	} else {
		// maybe global rate limit has it
		now := time.Now()
		until := time.Unix(0, atomic.LoadInt64(l.global))

		if until.After(now) {
			sleep = until.Sub(now)
		}
	}

	if sleep > 0 {
		select {
		case <-ctx.Done():
			b.lock.Unlock()
			return ctx.Err()
		case <-time.After(sleep):
		}
	}

	if b.remaining > 0 {
		b.remaining--
	}

	return nil
}

// Release releases the URL from the locks. This doesn't need a context for
// timing out, it doesn't block that much.
func (l *Limiter) Release(path string, headers http.Header) error {
	b := l.getBucket(path, false)
	if b == nil {
		return nil
	}

	defer b.lock.Unlock()

	// Check custom limiter
	if b.custom != nil {
		now := time.Now()

		if now.Sub(b.lastReset) >= b.custom.Reset {
			b.lastReset = now
			b.reset = now.Add(b.custom.Reset)
		}

		return nil
	}

	// Check if headers is nil or not:
	if headers == nil {
		return nil
	}

	var (
		// boolean
		global = headers.Get("X-RateLimit-Global")

		// seconds
		remaining  = headers.Get("X-RateLimit-Remaining")
		reset      = headers.Get("X-RateLimit-Reset")
		retryAfter = headers.Get("Retry-After")
	)

	switch {
	case retryAfter != "":
		i, err := strconv.Atoi(retryAfter)
		if err != nil {
			return errors.Wrap(err, "Invalid retryAfter "+retryAfter)
		}

		at := time.Now().Add(time.Duration(i) * time.Millisecond)

		if global != "" { // probably true
			atomic.StoreInt64(l.global, at.UnixNano())
		} else {
			b.reset = at
		}

	case reset != "":
		unix, err := strconv.ParseFloat(reset, 64)
		if err != nil {
			return errors.Wrap(err, "Invalid reset "+reset)
		}

		b.reset = time.Unix(0, int64(unix*float64(time.Second))).
			Add(ExtraDelay)
	}

	if remaining != "" {
		u, err := strconv.ParseUint(remaining, 10, 64)
		if err != nil {
			return errors.Wrap(err, "Invalid remaining "+remaining)
		}

		b.remaining = u
	}

	return nil
}
