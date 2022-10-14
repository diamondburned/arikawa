package cmdroute

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

type ctxKey uint8

const (
	_ ctxKey = iota
	ctxCtx
	deferTicketCtx
)

// UseContext returns a middleware that override the handler context to the
// given context. This middleware should only be used once in the parent-most
// router.
func UseContext(ctx context.Context) Middleware {
	return func(next InteractionHandler) InteractionHandler {
		return InteractionHandlerFunc(func(_ context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			return next.HandleInteraction(ctx, ev)
		})
	}
}

// FollowUpSender is a type that can send follow-up messages. Usually, anything
// that extends *api.Client can be used as a FollowUpSender.
type FollowUpSender interface {
	FollowUpInteraction(appID discord.AppID, token string, data api.InteractionResponseData) (*discord.Message, error)
}

// DeferOpts is the options for Deferrable().
type DeferOpts struct {
	// Timeout is the timeout for the handler to return a response. If the
	// handler does not return within this timeout, then it is deferred.
	//
	// Defaults to 1.5 seconds.
	Timeout time.Duration
	// Flags is the flags to set on the response.
	Flags discord.MessageFlags
	// Error is called when a follow-up message fails to send. If nil, it does
	// nothing.
	Error func(err error)
	// Done is called when the handler is done. If nil, it does nothing.
	Done func(*discord.Message)
}

// Deferrable marks a router as deferrable, meaning if the handler does not
// return a response within the deadline, the response will be automatically
// deferred.
func Deferrable(client FollowUpSender, opts DeferOpts) Middleware {
	if opts.Timeout == 0 {
		opts.Timeout = 1*time.Second + 500*time.Millisecond
	}

	return func(next InteractionHandler) InteractionHandler {
		return InteractionHandlerFunc(func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			timeout, cancel := context.WithTimeout(ctx, opts.Timeout)
			defer cancel()

			respCh := make(chan *api.InteractionResponse, 1)
			go func() {
				ctx := context.WithValue(ctx, deferTicketCtx, DeferTicket{
					ctx:     timeout,
					deferFn: cancel,
				})

				resp := next.HandleInteraction(ctx, ev)
				if resp != nil && opts.Flags > 0 {
					if resp.Data != nil {
						resp.Data.Flags = opts.Flags
					} else {
						resp.Data = &api.InteractionResponseData{
							Flags: opts.Flags,
						}
					}
				}

				respCh <- resp
			}()

			select {
			case resp := <-respCh:
				return resp
			case <-timeout.Done():
				go func() {
					resp := <-respCh
					if resp == nil || resp.Data == nil {
						return
					}
					m, err := client.FollowUpInteraction(ev.AppID, ev.Token, *resp.Data)
					if err != nil && opts.Error != nil {
						opts.Error(err)
					}
					if m != nil && opts.Done != nil {
						opts.Done(m)
					}
				}()
				return &api.InteractionResponse{
					Type: api.DeferredMessageInteractionWithSource,
					Data: &api.InteractionResponseData{
						Flags: opts.Flags,
					},
				}
			}
		})
	}
}

// DeferTicket is a ticket that can be used to defer a slash command. It can be
// used to manually send a response later.
type DeferTicket struct {
	ctx     context.Context
	deferFn context.CancelFunc
}

// DeferTicketFromContext returns the DeferTicket from the context. If no ticket
// is found, it returns a zero-value ticket.
func DeferTicketFromContext(ctx context.Context) DeferTicket {
	ticket, _ := ctx.Value(deferTicketCtx).(DeferTicket)
	return ticket
}

// IsDeferred returns true if the handler has been deferred.
func (t DeferTicket) IsDeferred() bool {
	return t.Context().Err() != nil
}

// Context returns the context that is done when the handler is deferred. If
// DeferTicket is zero-value, it returns the background context.
func (t DeferTicket) Context() context.Context {
	if t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

// Defer defers the response. If DeferTicket is zero-value, it does nothing.
func (t DeferTicket) Defer() {
	t.deferFn()
}
