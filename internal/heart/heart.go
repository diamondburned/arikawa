// Package heart implements a general purpose pacemaker.
package heart

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

// Debug is the default logger that Pacemaker uses.
var Debug = func(v ...interface{}) {}

var ErrDead = errors.New("no heartbeat replied")

// AtomicTime is a thread-safe UnixNano timestamp guarded by atomic.
type AtomicTime struct {
	unixnano int64
}

func (t *AtomicTime) Get() int64 {
	return atomic.LoadInt64(&t.unixnano)
}

func (t *AtomicTime) Set(time time.Time) {
	atomic.StoreInt64(&t.unixnano, time.UnixNano())
}

func (t *AtomicTime) Time() time.Time {
	return time.Unix(0, t.Get())
}

type Pacemaker struct {
	// Heartrate is the received duration between heartbeats.
	Heartrate time.Duration

	ticker time.Ticker
	Ticks  <-chan time.Time

	// Time in nanoseconds, guarded by atomic read/writes.
	SentBeat AtomicTime
	EchoBeat AtomicTime

	// Any callback that returns an error will stop the pacer.
	Pacer func(context.Context) error
}

func NewPacemaker(heartrate time.Duration, pacer func(context.Context) error) Pacemaker {
	p := Pacemaker{
		Heartrate: heartrate,
		Pacer:     pacer,
		ticker:    *time.NewTicker(heartrate),
	}
	p.Ticks = p.ticker.C
	// Reset states to its old position.
	now := time.Now()
	p.EchoBeat.Set(now)
	p.SentBeat.Set(now)

	return p
}

func (p *Pacemaker) Echo() {
	// Swap our received heartbeats
	p.EchoBeat.Set(time.Now())
}

// Dead, if true, will have Pace return an ErrDead.
func (p *Pacemaker) Dead() bool {
	var (
		echo = p.EchoBeat.Get()
		sent = p.SentBeat.Get()
	)

	if echo == 0 || sent == 0 {
		return false
	}

	return sent-echo > int64(p.Heartrate)*2
}

// Stop stops the pacemaker, or it does nothing if the pacemaker is not started.
func (p *Pacemaker) Stop() {
	p.ticker.Stop()
}

// pace sends a heartbeat with the appropriate timeout for the context.
func (p *Pacemaker) Pace() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.Heartrate)
	defer cancel()

	return p.PaceCtx(ctx)
}

func (p *Pacemaker) PaceCtx(ctx context.Context) error {
	if err := p.Pacer(ctx); err != nil {
		return err
	}

	p.SentBeat.Set(time.Now())

	if p.Dead() {
		return ErrDead
	}

	return nil
}

// func (p *Pacemaker) start() error {
// 	// Reset states to its old position.
// 	p.EchoBeat.Set(time.Time{})
// 	p.SentBeat.Set(time.Time{})

// 	// Create a new ticker.
// 	tick := time.NewTicker(p.Heartrate)
// 	defer tick.Stop()

// 	// Echo at least once
// 	p.Echo()

// 	for {
// 		if err := p.pace(); err != nil {
// 			return errors.Wrap(err, "failed to pace")
// 		}

// 		// Paced, save:
// 		p.SentBeat.Set(time.Now())

// 		if p.Dead() {
// 			return ErrDead
// 		}

// 		select {
// 		case <-p.stop:
// 			return nil

// 		case <-tick.C:
// 		}
// 	}
// }

// // StartAsync starts the pacemaker asynchronously. The WaitGroup is optional.
// func (p *Pacemaker) StartAsync(wg *sync.WaitGroup) (death chan error) {
// 	p.death = make(chan error)
// 	p.stop = make(chan struct{})
// 	p.once = sync.Once{}

// 	if wg != nil {
// 		wg.Add(1)
// 	}

// 	go func() {
// 		p.death <- p.start()
// 		// Debug.
// 		Debug("Pacemaker returned.")

// 		// Mark the pacemaker loop as done.
// 		if wg != nil {
// 			wg.Done()
// 		}
// 	}()

// 	return p.death
// }
