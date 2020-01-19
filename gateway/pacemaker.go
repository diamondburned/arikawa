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
	LastBeat [2]time.Time

	// Any callback that returns an error will stop the pacer.
	Pace func() error
	// Event
	OnDead func() error

	stop chan<- struct{}
}

func (p *Pacemaker) Echo() {
	// Swap our received heartbeats
	p.LastBeat[0], p.LastBeat[1] = time.Now(), p.LastBeat[0]
}

// Dead, if true, will have Pace return an ErrDead.
func (p *Pacemaker) Dead() bool {
	if p.LastBeat[0].IsZero() || p.LastBeat[1].IsZero() {
		return false
	}

	return p.LastBeat[0].Sub(p.LastBeat[1]) > p.Heartrate*2
}

func (p *Pacemaker) Stop() {
	if p.stop != nil {
		close(p.stop)
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

			if p.Dead() {
				if err := p.OnDead(); err != nil {
					return err
				}
			}
		}
	}
}

func (p *Pacemaker) StartAsync() (death <-chan error) {
	var ch = make(chan error)

	stop := make(chan struct{})
	p.stop = stop

	go func() {
		ch <- p.start(stop)
	}()

	return ch
}
