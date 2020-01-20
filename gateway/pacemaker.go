package gateway

import (
	"time"

	"github.com/pkg/errors"
)

var ErrDead = errors.New("no heartbeat replied")

type Pacemaker struct {
	// Heartrate is the received duration between heartbeats.
	Heartrate time.Duration

	// LastBeat logs the received heartbeats, with the newest one
	// first.
	// LastBeat [2]time.Time

	SentBeat time.Time
	EchoBeat time.Time

	// Any callback that returns an error will stop the pacer.
	Pace func() error
	// Event
	OnDead func() error

	stop  chan<- struct{}
	death chan error
}

func (p *Pacemaker) Echo() {
	// Swap our received heartbeats
	// p.LastBeat[0], p.LastBeat[1] = time.Now(), p.LastBeat[0]
	p.EchoBeat = time.Now()
}

// Dead, if true, will have Pace return an ErrDead.
func (p *Pacemaker) Dead() bool {
	/* Deprecated
	if p.LastBeat[0].IsZero() || p.LastBeat[1].IsZero() {
		return false
	}

	return p.LastBeat[0].Sub(p.LastBeat[1]) > p.Heartrate*2
	*/

	if p.EchoBeat.IsZero() || p.SentBeat.IsZero() {
		return false
	}

	return p.SentBeat.Sub(p.EchoBeat) > p.Heartrate*2
}

func (p *Pacemaker) Stop() {
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
}

// Start beats until it's dead.
func (p *Pacemaker) Start() error {
	stop := make(chan struct{})
	p.stop = stop

	return p.start(stop)
}

func (p *Pacemaker) start(stop chan struct{}) error {
	tick := time.NewTicker(p.Heartrate)
	defer tick.Stop()

	// Echo at least once
	p.Echo()

	for {
		select {
		case <-stop:
			return nil

		case <-tick.C:
			if err := p.Pace(); err != nil {
				return err
			}

			// Paced, save
			p.SentBeat = time.Now()

			if p.Dead() {
				return ErrDead
			}
		}
	}
}

func (p *Pacemaker) StartAsync() (death chan error) {
	p.death = make(chan error)

	stop := make(chan struct{})
	p.stop = stop

	go func() {
		p.death <- p.start(stop)
	}()

	return p.death
}
