// Package ophandler provides an Op channel reader that redistributes the events
// into handlers.
package ophandler

import (
	"context"

	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

// Loop starts a background goroutine that starts reading from src and
// distributes received events into the given handler. It's stopped once src is
// closed. The returned channel will be closed once src is closed.
func Loop(src <-chan ws.Op, dst *handler.Handler) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for op := range src {
			dst.Call(op.Data)
		}
		close(done)
	}()
	return done
}

// WaitForDone waits for the done channel returned by Loop until the channel is
// closed or the context expires.
func WaitForDone(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
