// Package heart implements a general purpose pacemaker.
package heart

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/utils/wsutil"
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

	// Time in nanoseconds, guarded by atomic read/writes.
	SentBeat AtomicTime
	EchoBeat AtomicTime

	// Any callback that returns an error will stop the pacer.
	Pace func() error

	stop  chan struct{}
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
	if p.stop != nil {
		p.stop <- struct{}{}
		Debug("(*Pacemaker).stop was sent a stop signal.")
	} else {
		Debug("(*Pacemaker).stop is nil, skipping.")
	}
}

func (p *Pacemaker) start() error {
	log.Println("HR:", p.Heartrate)
	tick := time.NewTicker(p.Heartrate)
	defer tick.Stop()

	// Echo at least once
	p.Echo()

	for {
		Debug("Pacemaker loop restarted.")

		if err := p.Pace(); err != nil {
			return err
		}

		Debug("Paced.")

		// Paced, save:
		p.SentBeat.Set(time.Now())

		if p.Dead() {
			return ErrDead
		}

		select {
		case <-p.stop:
			Debug("Received stop signal.")
			return nil

		case <-tick.C:
			Debug("Ticked. Restarting.")
		}
	}
}

// StartAsync starts the pacemaker asynchronously. The WaitGroup is optional.
func (p *Pacemaker) StartAsync(wg *sync.WaitGroup) (death chan error) {
	p.death = make(chan error)
	p.stop = make(chan struct{})

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		p.death <- p.start()
		// Debug.
		Debug("Pacemaker returned.")
		// Mark the stop channel as nil, so later Close() calls won't block forever.
		p.stop = nil

		// Mark the pacemaker loop as done.
		if wg != nil {
			wg.Done()
		}
	}()

	return p.death
}

// TODO API
type EventLoop interface {
	Heartbeat() error
	HandleEvent(ev wsutil.Event) error
}

// PacemakerLoop provides an event loop with a pacemaker.
type PacemakerLoop struct {
	pacemaker *Pacemaker // let's not copy this
	pacedeath chan error

	events  <-chan wsutil.Event
	handler func(wsutil.Event) error

	ErrorLog func(error)
}

func NewLoop(heartrate time.Duration, evs <-chan wsutil.Event, evl EventLoop) *PacemakerLoop {
	pacemaker := NewPacemaker(heartrate, evl.Heartbeat)

	return &PacemakerLoop{
		pacemaker: pacemaker,
		events:    evs,
		handler:   evl.HandleEvent,
	}
}

func (p *PacemakerLoop) errorLog(err error) {
	if p.ErrorLog == nil {
		Debug("Uncaught error:", err)
		return
	}

	p.ErrorLog(err)
}

func (p *PacemakerLoop) Echo() {
	p.pacemaker.Echo()
}

func (p *PacemakerLoop) Stop() {
	p.pacemaker.Stop()
}

func (p *PacemakerLoop) Stopped() bool {
	return p.pacedeath == nil
}

func (p *PacemakerLoop) Run() error {
	// If the event loop is already running.
	if p.pacedeath != nil {
		return nil
	}
	// callers should explicitly handle waitgroups.
	p.pacedeath = p.pacemaker.StartAsync(nil)

	defer func() {
		// mark pacedeath once done
		p.pacedeath = nil

		Debug("Pacemaker loop has exited.")
	}()

	for {
		select {
		case err := <-p.pacedeath:
			// Got a paceDeath, we're exiting from here on out.
			p.pacedeath = nil // mark

			if err == nil {
				// No error, just exit normally.
				return nil
			}

			return errors.Wrap(err, "Pacemaker died, reconnecting")

		case ev, ok := <-p.events:
			if !ok {
				// Events channel is closed. Kill the pacemaker manually and
				// die.
				p.pacemaker.Stop()
				return <-p.pacedeath
			}

			// Handle the event
			if err := p.handler(ev); err != nil {
				p.errorLog(errors.Wrap(err, "WS handler error"))
			}
		}
	}
}
