// Package handleloop provides clean abstractions to handle listening to
// channels and passing them onto event handlers.
package handleloop

import "github.com/diamondburned/arikawa/v3/utils/handler"

// Loop provides a reusable event looper abstraction. It is thread-safe to use
// concurrently.
type Loop struct {
	dst  *handler.Handler
	run  chan struct{}
	stop chan struct{}
}

func NewLoop(dst *handler.Handler) *Loop {
	return &Loop{
		dst:  dst,
		run:  make(chan struct{}, 1), // intentional 1 buffer
		stop: make(chan struct{}),    // intentional unbuffer
	}
}

// Start starts a new event loop. It will try to stop existing loops before.
func (l *Loop) Start(src <-chan interface{}) {
	// Ensure we're stopped.
	l.Stop()

	// Mark that we're running.
	l.run <- struct{}{}

	go func() {
		for {
			select {
			case event := <-src:
				l.dst.Call(event)

			case <-l.stop:
				l.stop <- struct{}{}
				return
			}
		}
	}()
}

// Stop tries to stop the Loop. If the Loop is not running, then it does
// nothing; thus, it can be called multiple times.
func (l *Loop) Stop() {
	// Ensure that we are running before stopping.
	select {
	case <-l.run:
		// running
	default:
		return
	}

	// send a close request
	l.stop <- struct{}{}
	// wait for a reply
	<-l.stop
}
