package wsutil

import (
	"time"

	"github.com/diamondburned/arikawa/utils/heart"
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
	return p == nil || p.pacedeath == nil
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

		WSDebug("Pacemaker loop has exited.")
	}()

	for {
		select {
		case err := <-p.pacedeath:
			return errors.Wrap(err, "Pacemaker died, reconnecting")

		case ev, ok := <-p.events:
			if !ok {
				// Events channel is closed. Kill the pacemaker manually and
				// die.
				p.pacemaker.Stop()
				return <-p.pacedeath
			}

			o, err := DecodeOP(ev)
			if err != nil {
				p.errorLog(errors.Wrap(err, "Failed to decode OP"))
				continue // ignore
			}

			// Check the events before handling.
			p.Extras.Check(o)

			// Handle the event
			if err := p.handler(o); err != nil {
				p.errorLog(errors.Wrap(err, "Handler failed"))
			}
		}
	}
}
