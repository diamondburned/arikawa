package cmdroute

import (
	"context"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
)

// Router is a router for slash commands. A zero-value Router is a valid router.
type Router struct {
	nodes  map[string]routeNode
	mws    []Middleware
	parent *Router  // parent router, if any
	groups []Router // next routers to check, if any
}

type routeNode interface {
	isRouteNode()
}

type routeNodeSub struct{ *Router }

type routeNodeCommand struct {
	command      CommandHandler
	autocomplete Autocompleter
}

type routeNodeComponent struct {
	component ComponentHandler
}

func (routeNodeSub) isRouteNode()       {}
func (routeNodeCommand) isRouteNode()   {}
func (routeNodeComponent) isRouteNode() {}

var _ webhook.InteractionHandler = (*Router)(nil)

// NewRouter creates a new Router.
func NewRouter() *Router {
	r := &Router{}
	r.init()
	return r
}

func (r *Router) init() {
	if r.nodes == nil {
		r.nodes = make(map[string]routeNode, 4)
	}
}

func (r *Router) add(name string, node routeNode) {
	r.init()

	_, ok := r.nodes[name]
	if ok {
		panic("cmdroute: node " + name + " already exists")
	}

	r.nodes[name] = node
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
	sub := NewRouter()
	sub.parent = r
	f(sub)

	r.add(name, routeNodeSub{sub})
}

// Add registers a slash command handler for the given command name.
func (r *Router) Add(name string, h CommandHandler) {
	r.add(name, routeNodeCommand{command: h})
}

// AddFunc is a convenience function that calls Handle with a
// CommandHandlerFunc.
func (r *Router) AddFunc(name string, f CommandHandlerFunc) {
	r.Add(name, f)
}

// Group creates a subrouter that handles certain commands within the parent
// command. This is useful for assigning middlewares to a group of commands that
// belong to the same parent command.
//
// For example, consider the following:
//
//	r := cmdroute.NewRouter()
//	r.Group(func(r *cmdroute.Router) {
//		r.Use(cmdroute.Deferrable(client, cmdroute.DeferOpts{}))
//		r.Add("foo", handleFoo)
//	})
//	r.Add("bar", handleBar)
//
// In this example, the middleware is only applied to the "foo" command, and not
// the "bar" command.
func (r *Router) Group(f func(r *Router)) {
	f(r.With())
}

// With is similar to Group, but it returns a new router instead of calling a
// function with a new router. This is useful for chaining middlewares once,
// such as:
//
//	r := cmdroute.NewRouter()
//	r.With(cmdroute.Deferrable(client, cmdroute.DeferOpts{})).Add("foo", handleFoo)
func (r *Router) With(mws ...Middleware) *Router {
	r.groups = append(r.groups, Router{})
	sub := &r.groups[len(r.groups)-1]
	sub.parent = r
	sub.mws = append(sub.mws, mws...)
	return sub
}

// HandleInteraction implements webhook.InteractionHandler. It only handles
// events of type CommandInteraction, otherwise nil is returned.
func (r *Router) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	switch data := ev.Data.(type) {
	case *discord.CommandInteraction:
		return r.HandleCommand(ev, data)
	case *discord.AutocompleteInteraction:
		return r.HandleAutocompletion(ev, data)
	case discord.ComponentInteraction:
		return r.handleComponent(ev, data)
	default:
		return nil
	}
}

func (r *Router) callHandler(ev *discord.InteractionEvent, fn InteractionHandlerFunc) *api.InteractionResponse {
	h := InteractionHandler(fn)

	// Apply middlewares, parent last, first one added last. This ensures that
	// when we call the handler, the middlewares are applied in the order they
	// were added.
	for r != nil {
		for i := len(r.mws) - 1; i >= 0; i-- {
			h = r.mws[i](h)
		}
		r = r.parent
	}

	return h.HandleInteraction(context.Background(), ev)
}

// HandleCommand implements CommandHandler. It applies middlewares onto the
// handler to be executed.
//
// Deprecated: This function should not be used directly. Use HandleInteraction
// instead.
func (r *Router) HandleCommand(ev *discord.InteractionEvent, data *discord.CommandInteraction) *api.InteractionResponse {
	cmdType := discord.SubcommandOptionType
	if cmdIsGroup(data) {
		cmdType = discord.SubcommandGroupOptionType
	}

	found, ok := r.findCommandHandler(ev, discord.CommandInteractionOption{
		Type:    cmdType,
		Name:    data.Name,
		Options: data.Options,
	})
	if !ok {
		return nil
	}

	return found.router.callCommandHandler(ev, found)
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

// findCommandHandler finds the command handler for the given command name.
// It checks the current router and its groups.
func (r *Router) findCommandHandler(ev *discord.InteractionEvent, data discord.CommandInteractionOption) (handlerData, bool) {
	found, ok := r.findCommandHandlerOnce(ev, data)
	if ok {
		return found, true
	}

	for _, sub := range r.groups {
		found, ok = sub.findCommandHandlerOnce(ev, data)
		if ok {
			return found, true
		}
	}

	return handlerData{}, false
}

// findCommandHandlerOnce finds the command handler for the given command name.
// It only checks the current router and not its groups.
func (r *Router) findCommandHandlerOnce(ev *discord.InteractionEvent, data discord.CommandInteractionOption) (handlerData, bool) {
	node, ok := r.nodes[data.Name]
	if !ok {
		return handlerData{}, false
	}

	switch node := node.(type) {
	case routeNodeSub:
		if len(data.Options) != 1 || data.Type != discord.SubcommandGroupOptionType {
			break
		}
		return node.findCommandHandler(ev, data.Options[0])
	case routeNodeCommand:
		if data.Type != discord.SubcommandOptionType {
			break
		}
		return handlerData{
			router:  r,
			handler: node.command,
			data:    data,
		}, true
	}

	return handlerData{}, false
}

func (r *Router) callCommandHandler(ev *discord.InteractionEvent, found handlerData) *api.InteractionResponse {
	return r.callHandler(ev,
		func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			data := found.handler.HandleCommand(ctx, CommandData{
				CommandInteractionOption: found.data,
				Event:                    ev,
				Data:                     ev.Data.(*discord.CommandInteraction),
			})
			if data == nil {
				return nil
			}

			return &api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: data,
			}
		},
	)
}

// AddAutocompleter registers an autocompleter for the given command name.
func (r *Router) AddAutocompleter(name string, ac Autocompleter) {
	r.init()

	node, ok := r.nodes[name].(routeNodeCommand)
	if !ok {
		panic("cmdroute: cannot add autocompleter to unknown command " + name)
	}

	node.autocomplete = ac
	r.nodes[name] = node
}

// AddAutocompleterFunc is a convenience function that calls AddAutocompleter
// with an AutocompleterFunc.
func (r *Router) AddAutocompleterFunc(name string, f AutocompleterFunc) {
	r.AddAutocompleter(name, f)
}

// HandleAutocompletion handles an autocompletion event.
//
// Deprecated: This function should not be used directly. Use HandleInteraction
// instead.
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

	return found.router.callAutocompletion(ev, found)
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

// findAutocompleter finds the autocomplete handler for the given option name.
// It checks the current router and its groups.
func (r *Router) findAutocompleter(ev *discord.InteractionEvent, data discord.AutocompleteOption) (autocompleterData, bool) {
	found, ok := r.findAutocompleterOnce(ev, data)
	if ok {
		return found, true
	}

	for _, sub := range r.groups {
		found, ok = sub.findAutocompleterOnce(ev, data)
		if ok {
			return found, true
		}
	}

	return autocompleterData{}, false
}

// findAutocompleter finds the autocomplete handler for the given option name.
// It only checks the current router and not its groups.
func (r *Router) findAutocompleterOnce(ev *discord.InteractionEvent, data discord.AutocompleteOption) (autocompleterData, bool) {
	node, ok := r.nodes[data.Name]
	if !ok {
		return autocompleterData{}, false
	}
	switch node := node.(type) {
	case routeNodeSub:
		if len(data.Options) != 1 || data.Type != discord.SubcommandGroupOptionType {
			break
		}
		for _, option := range data.Options {
			found, ok := node.findAutocompleter(ev, option)
			if ok {
				return found, true
			}
		}
	case routeNodeCommand:
		if node.autocomplete == nil {
			break
		}
		if data.Type != discord.SubcommandOptionType {
			break
		}
		return autocompleterData{
			router:  r,
			handler: node.autocomplete,
			data:    data,
		}, true
	}
	return autocompleterData{}, false
}

func (r *Router) callAutocompletion(ev *discord.InteractionEvent, found autocompleterData) *api.InteractionResponse {
	return r.callHandler(ev,
		func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			choices := found.handler.Autocomplete(ctx, AutocompleteData{
				AutocompleteOption: found.data,
				Event:              ev,
				Data:               ev.Data.(*discord.AutocompleteInteraction),
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

// AddComponent registers a component handler for the given component ID.
func (r *Router) AddComponent(id string, f ComponentHandler) {
	r.add(id, routeNodeComponent{f})
}

// AddComponentFunc is a convenience function that calls Handle with a
// ComponentHandlerFunc.
func (r *Router) AddComponentFunc(id string, f ComponentHandlerFunc) {
	r.AddComponent(id, f)
}

func (r *Router) handleComponent(ev *discord.InteractionEvent, component discord.ComponentInteraction) *api.InteractionResponse {
	node, ok := r.nodes[string(component.ID())].(routeNodeComponent)
	if ok {
		return r.callComponentHandler(ev, node.component)
	}
	return nil
}

func (r *Router) callComponentHandler(ev *discord.InteractionEvent, handler ComponentHandler) *api.InteractionResponse {
	return r.callHandler(ev,
		func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
			return handler.HandleComponent(ctx, ComponentData{
				Event:                ev,
				ComponentInteraction: ev.Data.(discord.ComponentInteraction),
			})
		},
	)
}
