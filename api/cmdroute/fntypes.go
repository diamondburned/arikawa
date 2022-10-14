package cmdroute

import (
	"context"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

// InteractionHandler is similar to webhook.InteractionHandler, but it also
// includes a context.
type InteractionHandler interface {
	// HandleInteraction is expected to return a response synchronously, either
	// to be followed-up later by deferring the response or to be responded
	// immediately.
	HandleInteraction(context.Context, *discord.InteractionEvent) *api.InteractionResponse
}

// InteractionHandlerFunc is a function that implements InteractionHandler.
type InteractionHandlerFunc func(context.Context, *discord.InteractionEvent) *api.InteractionResponse

var _ InteractionHandler = InteractionHandlerFunc(nil)

// HandleInteraction implements InteractionHandler.
func (f InteractionHandlerFunc) HandleInteraction(ctx context.Context, e *discord.InteractionEvent) *api.InteractionResponse {
	return f(ctx, e)
}

// Middleware is a function type that wraps a Handler. It can be used as a
// middleware for the handler.
type Middleware = func(next InteractionHandler) InteractionHandler

/*
 * Command
 */

// CommandData is passed to a CommandHandler's HandleCommand method.
type CommandData struct {
	discord.CommandInteractionOption
	Event *discord.InteractionEvent
}

// CommandHandler is a slash command handler.
type CommandHandler interface {
	// HandleCommand is expected to return a response synchronously, either to
	// be followed-up later by deferring the response or to be responded
	// immediately.
	//
	// All HandleCommand invocations are given a 3-second deadline. If the
	// handler does not return a response within the deadline, the response will
	// be automatically deferred in a goroutine, and the returned response will
	// be sent to the user through the API instead.
	HandleCommand(ctx context.Context, data CommandData) *api.InteractionResponseData
}

// CommandHandlerFunc is a function that implements CommandHandler.
type CommandHandlerFunc func(ctx context.Context, data CommandData) *api.InteractionResponseData

var _ CommandHandler = CommandHandlerFunc(nil)

// HandleCommand implements CommandHandler.
func (f CommandHandlerFunc) HandleCommand(ctx context.Context, data CommandData) *api.InteractionResponseData {
	return f(ctx, data)
}

/*
 * Autocomplete
 */

// AutocompleteData is passed to an Autocompleter's Autocomplete method.
type AutocompleteData struct {
	discord.AutocompleteOption
	Event *discord.InteractionEvent
}

// Autocompleter is a type for an autocompleter.
type Autocompleter interface {
	// Autocomplete is expected to return a list of choices synchronously.
	// If nil is returned, then no responses will be sent. The function must
	// return an empty slice if there are no choices.
	Autocomplete(ctx context.Context, data AutocompleteData) api.AutocompleteChoices
}

// AutocompleterFunc is a function that implements the Autocompleter interface.
type AutocompleterFunc func(ctx context.Context, data AutocompleteData) api.AutocompleteChoices

var _ Autocompleter = (AutocompleterFunc)(nil)

// Autocomplete implements webhook.InteractionHandler.
func (f AutocompleterFunc) Autocomplete(ctx context.Context, data AutocompleteData) api.AutocompleteChoices {
	return f(ctx, data)
}
