package rate

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/pkg/errors"
)

// ExtraDelay because Discord is trash. I've seen this in both litcord and
// discordgo, with dgo claiming from  experiments.
// RE: Those who want others to fix it for them: release the source code then.
const ExtraDelay = 250 * time.Millisecond

// ErrTimedOutEarly is the error returned by Limiter.Acquire, if a rate limit
// exceeds the deadline of the context.Context or api.AcquireOptions.DontWait
// is set to true
var ErrTimedOutEarly = errors.New(
	"rate: rate limit exceeds context deadline or is blocked acquire options")

// This makes me suicidal.
// https://github.com/bwmarrin/discordgo/blob/master/ratelimit.go

type Limiter struct {
	// Only 1 per bucket
	CustomLimits []*CustomRateLimit

	Prefix string

	// global is a pointer to prevent ARM-compatibility alignment.
	global *int64 // atomic guarded, unixnano

	bucketMu sync.Mutex
	buckets  map[string]*bucket
}

type CustomRateLimit struct {
	Contains string
	Reset    time.Duration
}

type contextKey uint8

const (
	// AcquireOptionsKey is the key used to store the AcquireOptions in the
	// context.
	acquireOptionsKey contextKey = iota
)

type AcquireOptions struct {
	// DontWait prevents rate.Limiters from waiting for a rate limit. Instead
	// they will return an rate.ErrTimedOutEarly.
	DontWait bool
}

// Context wraps the given ctx to have the AcquireOptions.
func (opts AcquireOptions) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, acquireOptionsKey, opts)
}

type bucket struct {
	lock   moreatomic.CtxMutex
	custom *CustomRateLimit

	remaining uint64

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
		buckets:      map[string]*bucket{},
		CustomLimits: []*CustomRateLimit{},
	}
}

func (l *Limiter) getBucket(path string, store bool) *bucket {
	path = ParseBucketKey(strings.TrimPrefix(path, l.Prefix))

	l.bucketMu.Lock()
	defer l.bucketMu.Unlock()

	bc, ok := l.buckets[path]
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

		l.buckets[path] = bc
		return bc
	}

	return bc
}

// Acquire acquires the rate limiter for the given URL bucket.
func (l *Limiter) Acquire(ctx context.Context, path string) error {
	var options AcquireOptions

	if untypedOptions := ctx.Value(acquireOptionsKey); untypedOptions != nil {
		// Zero value are default anyways, so we can ignore ok.
		options, _ = untypedOptions.(AcquireOptions)
	}

	b := l.getBucket(path, true)

	if err := b.lock.Lock(ctx); err != nil {
		return err
	}

	// Deadline until the limiter is released.
	until := time.Time{}
	now := time.Now()

	if b.remaining == 0 && b.reset.After(now) {
		// out of turns, gotta wait
		until = b.reset
	} else {
		// maybe global rate limit has it
		until = time.Unix(0, atomic.LoadInt64(l.global))
	}

	if until.After(now) {
		if options.DontWait {
			return ErrTimedOutEarly
		} else if deadline, ok := ctx.Deadline(); ok && until.After(deadline) {
			return ErrTimedOutEarly
		}

		select {
		case <-ctx.Done():
			b.lock.Unlock()
			return ctx.Err()
		case <-time.After(until.Sub(now)):
		}
	}

	if b.remaining > 0 {
		b.remaining--
	}

	return nil
}

// Release releases the URL from the locks. This doesn't need a context for
// timing out, since it doesn't block that much.
func (l *Limiter) Release(path string, headers http.Header) error {
	b := l.getBucket(path, false)
	if b == nil {
		return nil
	}

	// TryUnlock because Release may be called when Acquire has not been.
	defer b.lock.TryUnlock()

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
		reset      = headers.Get("X-RateLimit-Reset") // float
		retryAfter = headers.Get("Retry-After")
	)

	switch {
	case retryAfter != "":
		i, err := strconv.Atoi(retryAfter)
		if err != nil {
			return errors.Wrapf(err, "invalid retryAfter %q", retryAfter)
		}

		at := time.Now().Add(time.Duration(i) * time.Second)

		if global != "" { // probably "true"
			atomic.StoreInt64(l.global, at.UnixNano())
		} else {
			b.reset = at
		}

	case reset != "":
		unix, err := strconv.ParseFloat(reset, 64)
		if err != nil {
			return errors.Wrap(err, "invalid reset "+reset)
		}

		sec := int64(unix)
		nsec := int64((unix - float64(sec)) * float64(time.Second))

		b.reset = time.Unix(sec, nsec).Add(ExtraDelay)
	}

	if remaining != "" {
		u, err := strconv.ParseUint(remaining, 10, 64)
		if err != nil {
			return errors.Wrap(err, "invalid remaining "+remaining)
		}

		b.remaining = u
	}

	return nil
}
