package wsutil

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/internal/heart"
	"github.com/diamondburned/arikawa/internal/moreatomic"
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
	running moreatomic.Bool

	stop    chan struct{}
	events  <-chan Event
	handler func(*OP) error

	stack []byte

	Extras ExtraHandlers

	ErrorLog func(error)
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

// Stop stops the pacer loop. It does nothing if the loop is already stopped.
func (p *PacemakerLoop) Stop() {
	if p.Stopped() {
		return
	}

	// Despite p.running and p.stop being thread-safe on their own, this entire
	// block is actually not thread-safe.
	p.Pacemaker.Stop()
	close(p.stop)
}

func (p *PacemakerLoop) Stopped() bool {
	return p == nil || !p.running.Get()
}

func (p *PacemakerLoop) RunAsync(
	heartrate time.Duration, evs <-chan Event, evl EventLoopHandler, exit func(error)) {

	WSDebug("Starting the pacemaker loop.")

	p.Pacemaker = heart.NewPacemaker(heartrate, evl.HeartbeatCtx)
	p.handler = evl.HandleOP
	p.events = evs
	p.stack = debug.Stack()
	p.stop = make(chan struct{})

	p.running.Set(true)

	go func() {
		exit(p.startLoop())
	}()
}

func (p *PacemakerLoop) startLoop() error {
	defer WSDebug("Pacemaker loop has exited.")
	defer p.running.Set(false)
	defer p.Pacemaker.Stop()

	for {
		select {
		case <-p.stop:
			WSDebug("Stop requested; exiting.")
			return nil

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
