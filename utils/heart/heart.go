// Package heart implements a general purpose pacemaker.
package heart

import (
	"sync"
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

type atomicStop atomic.Value

func (s *atomicStop) Stop() bool {
	if v := (*atomic.Value)(s).Load(); v != nil {
		ch := v.(chan struct{})
		close(ch)
		return true
	}
	return false
}
func (s *atomicStop) Recv() <-chan struct{} {
	if v := (*atomic.Value)(s).Load(); v != nil {
		return v.(chan struct{})
	}
	return nil
}
func (s *atomicStop) SetNil() {
	(*atomic.Value)(s).Store((chan struct{})(nil))
}
func (s *atomicStop) Reset() {
	(*atomic.Value)(s).Store(make(chan struct{}))
}

type Pacemaker struct {
	// Heartrate is the received duration between heartbeats.
	Heartrate time.Duration

	// Time in nanoseconds, guarded by atomic read/writes.
	SentBeat AtomicTime
	EchoBeat AtomicTime

	// Any callback that returns an error will stop the pacer.
	Pace func() error

	stop  atomicStop
	death chan error
}

func NewPacemaker(heartrate time.Duration, pacer func() error) *Pacemaker {
	return &Pacemaker{
		Heartrate: heartrate,
		Pace:      pacer,
	}
}

func (p *Pacemaker) Echo() {
	// Swap our received heartbeats
	// p.LastBeat[0], p.LastBeat[1] = time.Now(), p.LastBeat[0]
	p.EchoBeat.Set(time.Now())
}

// Dead, if true, will have Pace return an ErrDead.
func (p *Pacemaker) Dead() bool {
	/* Deprecated
	if p.LastBeat[0].IsZero() || p.LastBeat[1].IsZero() {
		return false
	}

	return p.LastBeat[0].Sub(p.LastBeat[1]) > p.Heartrate*2
	*/

	var (
		echo = p.EchoBeat.Get()
		sent = p.SentBeat.Get()
	)

	if echo == 0 || sent == 0 {
		return false
	}

	return sent-echo > int64(p.Heartrate)*2
}

func (p *Pacemaker) Stop() {
	if p.stop.Stop() {
		Debug("(*Pacemaker).stop was sent a stop signal.")
	} else {
		Debug("(*Pacemaker).stop is nil, skipping.")
	}
}

func (p *Pacemaker) start() error {
	// Reset states to its old position.
	p.EchoBeat.Set(time.Time{})
	p.SentBeat.Set(time.Time{})

	// Create a new ticker.
	tick := time.NewTicker(p.Heartrate)
	defer tick.Stop()

	// Echo at least once
	p.Echo()

	for {

		if err := p.Pace(); err != nil {
			return err
		}

		// Paced, save:
		p.SentBeat.Set(time.Now())

		if p.Dead() {
			return ErrDead
		}

		select {
		case <-p.stop.Recv():
			return nil

		case <-tick.C:
		}
	}
}

// StartAsync starts the pacemaker asynchronously. The WaitGroup is optional.
func (p *Pacemaker) StartAsync(wg *sync.WaitGroup) (death chan error) {
	p.death = make(chan error)
	p.stop.Reset()

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		p.death <- p.start()
		// Debug.
		Debug("Pacemaker returned.")
		// Mark the stop channel as nil, so later Close() calls won't block forever.
		p.stop.SetNil()

		// Mark the pacemaker loop as done.
		if wg != nil {
			wg.Done()
		}
	}()

	return p.death
}
