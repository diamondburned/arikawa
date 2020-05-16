package wsutil

import (
	"time"

	"github.com/diamondburned/arikawa/utils/heart"
	"github.com/diamondburned/arikawa/utils/moreatomic"
	"github.com/pkg/errors"
)

// TODO API
type EventLoop interface {
	Heartbeat() error
	HandleOP(*OP) error
	// HandleEvent(ev Event) error
}

// PacemakerLoop provides an event loop with a pacemaker.
type PacemakerLoop struct {
	pacemaker *heart.Pacemaker // let's not copy this
	pacedeath chan error

	running moreatomic.Bool

	events  <-chan Event
	handler func(*OP) error

	Extras ExtraHandlers

	ErrorLog func(error)
}

func NewLoop(heartrate time.Duration, evs <-chan Event, evl EventLoop) *PacemakerLoop {
	pacemaker := heart.NewPacemaker(heartrate, evl.Heartbeat)

	return &PacemakerLoop{
		pacemaker: pacemaker,
		events:    evs,
		handler:   evl.HandleOP,
	}
}

func (p *PacemakerLoop) errorLog(err error) {
	if p.ErrorLog == nil {
		WSDebug("Uncaught error:", err)
		return
	}

	p.ErrorLog(err)
}

func (p *PacemakerLoop) Pace() error {
	return p.pacemaker.Pace()
}

func (p *PacemakerLoop) Echo() {
	p.pacemaker.Echo()
}

func (p *PacemakerLoop) Stop() {
	p.pacemaker.Stop()
}

func (p *PacemakerLoop) Stopped() bool {
	return p == nil || !p.running.Get()
}

func (p *PacemakerLoop) RunAsync(exit func(error)) {
	WSDebug("Starting the pacemaker loop.")

	// callers should explicitly handle waitgroups.
	p.pacedeath = p.pacemaker.StartAsync(nil)
	p.running.Set(true)

	go func() {
		exit(p.startLoop())
	}()
}

func (p *PacemakerLoop) startLoop() error {
	defer WSDebug("Pacemaker loop has exited.")
	defer p.running.Set(false)

	for {
		select {
		case err := <-p.pacedeath:
			WSDebug("Pacedeath returned with error:", err)
			return errors.Wrap(err, "pacemaker died, reconnecting")

		case ev, ok := <-p.events:
			if !ok {
				WSDebug("Events channel closed, stopping pacemaker.")
				defer WSDebug("Pacemaker stopped automatically.")
				// Events channel is closed. Kill the pacemaker manually and
				// die.
				p.pacemaker.Stop()
				return <-p.pacedeath
			}

			o, err := DecodeOP(ev)
			if err != nil {
				return errors.Wrap(err, "failed to decode OP")
			}

			// Check the events before handling.
			p.Extras.Check(o)

			// Handle the event
			if err := p.handler(o); err != nil {
				p.errorLog(errors.Wrap(err, "handler failed"))
			}
		}
	}
}
