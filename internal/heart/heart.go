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

// AtomicDuration is a thread-safe Duration guarded by atomic.
type AtomicDuration struct {
	duration int64
}

func (d *AtomicDuration) Get() time.Duration {
	return time.Duration(atomic.LoadInt64(&d.duration))
}

func (d *AtomicDuration) Set(dura time.Duration) {
	atomic.StoreInt64(&d.duration, int64(dura))
}

// Pacemaker is the internal pacemaker state. All fields are not thread-safe
// unless they're atomic.
type Pacemaker struct {
	// Heartrate is the received duration between heartbeats.
	Heartrate AtomicDuration

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
		Heartrate: AtomicDuration{int64(heartrate)},
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

	return sent-echo > int64(p.Heartrate.Get())*2
}

// SetHeartRate sets the ticker's heart rate.
func (p *Pacemaker) SetPace(heartrate time.Duration) {
	p.Heartrate.Set(heartrate)

	// To uncomment when 1.16 releases and we drop support for 1.14.
	// p.ticker.Reset(heartrate)

	p.ticker.Stop()
	p.ticker = *time.NewTicker(heartrate)
	p.Ticks = p.ticker.C
}

// Stop stops the pacemaker, or it does nothing if the pacemaker is not started.
func (p *Pacemaker) StopTicker() {
	p.ticker.Stop()
}

// pace sends a heartbeat with the appropriate timeout for the context.
func (p *Pacemaker) Pace() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.Heartrate.Get())
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
