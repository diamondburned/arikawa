package wsutil

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/internal/heart"
	"github.com/pkg/errors"
)

type errBrokenConnection struct {
	underneath error
}

// Error formats the broken connection error with the message "explicit
// connection break."
func (err errBrokenConnection) Error() string {
	return "explicit connection break: " + err.underneath.Error()
}

// Unwrap returns the underlying error.
func (err errBrokenConnection) Unwrap() error {
	return err.underneath
}

// ErrBrokenConnection marks the given error as a broken connection error. This
// error will cause the pacemaker loop to break and return the error. The error,
// when stringified, will say "explicit connection break."
func ErrBrokenConnection(err error) error {
	return errBrokenConnection{underneath: err}
}

// IsBrokenConnection returns true if the error is a broken connection error.
func IsBrokenConnection(err error) bool {
	var broken *errBrokenConnection
	return errors.As(err, &broken)
}

// TODO API
type EventLoopHandler interface {
	EventHandler
	HeartbeatCtx(context.Context) error
}

// PacemakerLoop provides an event loop with a pacemaker. A zero-value instance
// is a valid instance only when RunAsync is called first.
type PacemakerLoop struct {
	heart.Pacemaker
	Extras   ExtraHandlers
	ErrorLog func(error)

	events  <-chan Event
	control chan func()
	handler func(*OP) error
}

func (p *PacemakerLoop) errorLog(err error) {
	if p.ErrorLog == nil {
		WSDebug("Uncaught error:", err)
		return
	}

	p.ErrorLog(err)
}

// Pace calls the pacemaker's Pace function.
func (p *PacemakerLoop) Pace(ctx context.Context) error {
	return p.Pacemaker.PaceCtx(ctx)
}

// StartBeating asynchronously starts the pacemaker loop.
func (p *PacemakerLoop) StartBeating(d time.Duration, evl EventLoopHandler, exit func(error)) {
	WSDebug("Starting the pacemaker loop.")

	p.Pacemaker = heart.NewPacemaker(d, evl.HeartbeatCtx)
	p.control = make(chan func())
	p.handler = evl.HandleOP

	go func() { exit(p.startLoop()) }()
}

// SetEventChannel sets the event channel inside the event loop. There is no
// guarantee that the channel is set when the function returns. This function is
// concurrently safe.
func (p *PacemakerLoop) SetEventChannel(evCh <-chan Event) {
	p.control <- func() { p.events = evCh }
}

func (p *PacemakerLoop) startLoop() error {
	defer WSDebug("Pacemaker loop has exited.")
	defer p.Pacemaker.StopTicker()

	for {
		select {
		case <-p.Pacemaker.Ticks:
			if err := p.Pacemaker.Pace(); err != nil {
				return errors.Wrap(err, "pace failed, reconnecting")
			}

		case ev, ok := <-p.events:
			if !ok {
				WSDebug("Events channel closed, stopping pacemaker.")
				return nil
			}

			if ev.Error != nil {
				return errors.Wrap(ev.Error, "event returned error")
			}

			o, err := DecodeOP(ev)
			if err != nil {
				return errors.Wrap(err, "failed to decode OP")
			}

			// Check the events before handling.
			p.Extras.Check(o)

			// Handle the event
			if err := p.handler(o); err != nil {
				if IsBrokenConnection(err) {
					return errors.Wrap(err, "handler failed")
				}

				p.errorLog(err)
			}
		}
	}
}
