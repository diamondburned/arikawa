package cmdroute

import (
	"context"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
)

// Router is a router for slash commands. A zero-value Router is a valid router.
type Router struct {
	nodes map[string]routeNode
	mws   []Middleware
	stack []*Router
}

type routeNode struct {
	sub *Router
	cmd CommandHandler
	com Autocompleter
}

var _ webhook.InteractionHandler = (*Router)(nil)

// NewRouter creates a new Router.
func NewRouter() *Router {
	r := &Router{}
	r.init()
	return r
}

func (r *Router) init() {
	if r.stack == nil {
		r.stack = []*Router{r}
	}
	if r.nodes == nil {
		r.nodes = make(map[string]routeNode, 4)
	}
}

// Use adds a middleware to the router. The middleware is applied to all
// subcommands and subrouters. Middlewares are applied in the order they are
// added, with the middlewares in the parent router being applied first.
func (r *Router) Use(mws ...Middleware) {
	r.init()
	r.mws = append(r.mws, mws...)
}

// Sub creates a subrouter that handles all subcommands that are under the
// parent command of the given name.
func (r *Router) Sub(name string, f func(r *Router)) {
	r.init()

	node, ok := r.nodes[name]
	if ok && node.sub == nil {
		panic("cmdroute: command " + name + " already exists")
	}

	sub := NewRouter()
	sub.stack = append(append([]*Router(nil), r.stack...), sub)
	f(sub)

	r.nodes[name] = routeNode{sub: sub}
}

// Add registers a slash command handler for the given command name.
func (r *Router) Add(name string, h CommandHandler) {
	r.init()

	node, ok := r.nodes[name]
	if ok {
		panic("cmdroute: command " + name + " already exists")
	}

	node.cmd = h
	r.nodes[name] = node
}

// AddFunc is a convenience function that calls Handle with a
// CommandHandlerFunc.
func (r *Router) AddFunc(name string, f CommandHandlerFunc) {
	r.Add(name, f)
}

// HandleInteraction implements webhook.InteractionHandler. It only handles
// events of type CommandInteraction, otherwise nil is returned.
func (r *Router) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	switch data := ev.Data.(type) {
	case *discord.CommandInteraction:
		return r.HandleCommand(ev, data)
	case *discord.AutocompleteInteraction:
		return r.HandleAutocompletion(ev, data)
	default:
		return nil
	}
}

func (r *Router) handleInteraction(ev *discord.InteractionEvent, fn InteractionHandlerFunc) *api.InteractionResponse {
	h := InteractionHandler(fn)

	// Apply middlewares, parent last, first one added last. This ensures that
	// when we call the handler, the middlewares are applied in the order they
	// were added.
	for i := len(r.stack) - 1; i >= 0; i-- {
		r := r.stack[i]
		for j := len(r.mws) - 1; j >= 0; j-- {
			h = r.mws[j](h)
		}
	}

	return h.HandleInteraction(context.Background(), ev)
}

// HandleCommand implements CommandHandler. It applies middlewares onto the
// handler to be executed.
func (r *Router) HandleCommand(ev *discord.InteractionEvent, data *discord.CommandInteraction) *api.InteractionResponse {
	cmdType := discord.SubcommandOptionType
	if cmdIsGroup(data) {
		cmdType = discord.SubcommandGroupOptionType
	}

	found, ok := r.findHandler(ev, discord.CommandInteractionOption{
		Type:    cmdType,
		Name:    data.Name,
		Options: data.Options,
	})
	if !ok {
		return nil
	}

	return found.router.handleCommand(ev, found)
}

func (r *Router) handleCommand(ev *discord.InteractionEvent, found handlerData) *api.InteractionResponse {
	return r.handleInteraction(ev,
		func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			resp := found.handler.HandleCommand(ctx, CommandData{
				CommandInteractionOption: found.data,
				Event:                    ev,
			})
			if resp == nil {
				return nil
			}

			return resp
		},
	)
}

func cmdIsGroup(data *discord.CommandInteraction) bool {
	for _, opt := range data.Options {
		switch opt.Type {
		case discord.SubcommandGroupOptionType, discord.SubcommandOptionType:
			return true
		}
	}
	return false
}

type handlerData struct {
	router  *Router
	handler CommandHandler
	data    discord.CommandInteractionOption
}

func (r *Router) findHandler(ev *discord.InteractionEvent, data discord.CommandInteractionOption) (handlerData, bool) {
	node, ok := r.nodes[data.Name]
	if !ok {
		return handlerData{}, false
	}

	switch {
	case node.sub != nil:
		if len(data.Options) != 1 || data.Type != discord.SubcommandGroupOptionType {
			break
		}
		return node.sub.findHandler(ev, data.Options[0])
	case node.cmd != nil:
		if data.Type != discord.SubcommandOptionType {
			break
		}
		return handlerData{
			router:  r,
			handler: node.cmd,
			data:    data,
		}, true
	}

	return handlerData{}, false
}

// AddAutocompleter registers an autocompleter for the given command name.
func (r *Router) AddAutocompleter(name string, ac Autocompleter) {
	r.init()

	node, ok := r.nodes[name]
	if !ok || node.cmd == nil {
		panic("cmdroute: command " + name + " does not exist or is not a (sub)command")
	}

	node.com = ac
	r.nodes[name] = node
}

// AddAutocompleterFunc is a convenience function that calls AddAutocompleter
// with an AutocompleterFunc.
func (r *Router) AddAutocompleterFunc(name string, f AutocompleterFunc) {
	r.AddAutocompleter(name, f)
}

// HandleAutocompletion handles an autocompletion event.
func (r *Router) HandleAutocompletion(ev *discord.InteractionEvent, data *discord.AutocompleteInteraction) *api.InteractionResponse {
	cmdType := discord.SubcommandOptionType
	if autocompIsGroup(data) {
		cmdType = discord.SubcommandGroupOptionType
	}

	found, ok := r.findAutocompleter(ev, discord.AutocompleteOption{
		Type:    cmdType,
		Name:    data.Name,
		Options: data.Options,
	})
	if !ok {
		return nil
	}

	return found.router.handleAutocompletion(ev, found)
}

func (r *Router) handleAutocompletion(ev *discord.InteractionEvent, found autocompleterData) *api.InteractionResponse {
	return r.handleInteraction(ev,
		func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			choices := found.handler.Autocomplete(ctx, AutocompleteData{
				AutocompleteOption: found.data,
				Event:              ev,
			})
			if choices == nil {
				return nil
			}

			return &api.InteractionResponse{
				Type: api.AutocompleteResult,
				Data: &api.InteractionResponseData{
					Choices: choices,
				},
			}
		},
	)
}

func autocompIsGroup(data *discord.AutocompleteInteraction) bool {
	for _, opt := range data.Options {
		switch opt.Type {
		case discord.SubcommandGroupOptionType, discord.SubcommandOptionType:
			return true
		}
	}
	return false
}

type autocompleterData struct {
	router  *Router
	handler Autocompleter
	data    discord.AutocompleteOption
}

func (r *Router) findAutocompleter(ev *discord.InteractionEvent, data discord.AutocompleteOption) (autocompleterData, bool) {
	node, ok := r.nodes[data.Name]
	if !ok {
		return autocompleterData{}, false
	}

	switch {
	case node.sub != nil:
		if len(data.Options) != 1 || data.Type != discord.SubcommandGroupOptionType {
			break
		}
		return node.sub.findAutocompleter(ev, data.Options[0])
	case node.com != nil:
		if data.Type != discord.SubcommandOptionType {
			break
		}
		return autocompleterData{
			router:  r,
			handler: node.com,
			data:    data,
		}, true
	}

	return autocompleterData{}, false
}
