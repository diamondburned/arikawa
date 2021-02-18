package bot

import (
	"reflect"

	"github.com/diamondburned/arikawa/v2/gateway"
)

// eventIntents maps event pointer types to intents.
var eventIntents = deriveAllIntents()

func deriveAllIntents() map[reflect.Type]gateway.Intents {
	eventIntents := make(map[reflect.Type]gateway.Intents, len(gateway.EventIntents))

	for event, intent := range gateway.EventIntents {
		fn, ok := gateway.EventCreator[event]
		if !ok {
			continue
		}

		eventIntents[reflect.TypeOf(fn())] = intent
	}

	return eventIntents
}

type command struct {
	value       reflect.Value // Func
	event       reflect.Type
	isInterface bool
}

func newCommand(value reflect.Value, event reflect.Type) command {
	return command{
		value:       value,
		event:       event,
		isInterface: event.Kind() == reflect.Interface,
	}
}

func (c *command) isEvent(t reflect.Type) bool {
	return (!c.isInterface && c.event == t) || (c.isInterface && t.Implements(c.event))
}

func (c *command) call(arg0 interface{}, argv ...reflect.Value) (interface{}, error) {
	return callWith(c.value, arg0, argv...)
}

// intents returns the command's intents from the event.
func (c *command) intents() gateway.Intents {
	intents, ok := eventIntents[c.event]
	if !ok {
		return 0
	}
	return intents
}

// isInteractable returns true if the command is either a MessageCreate or
// InteractionCreate command.
func (c *command) isInteractable() bool {
	return c.event == typeMessageCreate || c.event == typeInteractionCreate
}

func callWith(caller reflect.Value, arg0 interface{}, argv ...reflect.Value) (interface{}, error) {
	var callargs = make([]reflect.Value, 0, 1+len(argv))

	if v, ok := arg0.(reflect.Value); ok {
		callargs = append(callargs, v)
	} else {
		callargs = append(callargs, reflect.ValueOf(arg0))
	}

	callargs = append(callargs, argv...)
	return errorReturns(caller.Call(callargs))
}

type caller interface {
	call(arg0 interface{}, argv ...reflect.Value) (interface{}, error)
}

func errorReturns(returns []reflect.Value) (interface{}, error) {
	// Handlers may return nothing.
	if len(returns) == 0 {
		return nil, nil
	}

	// assume first return is always error, since we checked for this in
	// parseCommands.
	v := returns[len(returns)-1].Interface()
	// If the last return (error) is nil.
	if v == nil {
		// If we only have 1 returns, that return must be the error. The error
		// is nil, so nil is returned.
		if len(returns) == 1 {
			return nil, nil
		}

		// Return the first argument as-is. The above returns[-1] check assumes
		// 2 return values (T, error), meaning returns[0] is the T value.
		return returns[0].Interface(), nil
	}

	// Treat the last return as an error.
	return nil, v.(error)
}

// MethodContext is an internal struct containing fields to make this library
// work. As such, they're all unexported. Description, however, is exported for
// editing, and may be used to generate more informative help messages.
type MethodContext struct {
	command
	method      reflect.Method // extend
	middlewares []*MiddlewareContext

	Description string

	// MethodName is the name of the method. This field should NOT be changed.
	MethodName string

	// Command is the Discord command used to call the method.
	Command string // plumb if empty

	// Aliases is alternative way to call command in Discord.
	Aliases []string

	// Hidden if true will not be shown by (*Subcommand).HelpGenerate().
	Hidden bool

	// Variadic is true if the function is a variadic one or if the last
	// argument accepts multiple strings.
	Variadic bool

	Arguments []Argument
}

func parseMethod(value reflect.Value, method reflect.Method) *MethodContext {
	methodT := value.Type()
	numArgs := methodT.NumIn()

	if numArgs == 0 {
		// Doesn't meet the requirement for an event, continue.
		return nil
	}

	// Check number of returns:
	numOut := methodT.NumOut()

	// Returns can either be:
	// Nothing                     - func()
	// An error                    - func() error
	// An error and something else - func() (T, error)
	if numOut > 2 {
		return nil
	}

	// Check the last return's type if the method returns anything.
	if numOut > 0 {
		if i := methodT.Out(numOut - 1); i == nil || !i.Implements(typeIError) {
			// Invalid, skip.
			return nil
		}
	}

	var command = MethodContext{
		command:    newCommand(value, methodT.In(0)),
		method:     method,
		MethodName: method.Name,
		Variadic:   methodT.IsVariadic(),
	}

	// Only set the command name if it's a MessageCreate handler.
	if command.isInteractable() {
		command.Command = lowerFirstLetter(command.method.Name)
	}

	if numArgs > 1 {
		// Event handlers that aren't MessageCreate or InteractionCreate should
		// not have arguments.
		if !command.isInteractable() {
			return nil
		}

		// If the event type is messageCreate:
		command.Arguments = make([]Argument, 0, numArgs-1)

		// Fill up arguments. This should work with cusP and manP
		for i := 1; i < numArgs; i++ {
			t := methodT.In(i)
			a, err := newArgument(t, command.Variadic)
			if err != nil {
				panic("error parsing argument " + t.String() + ": " + err.Error())
			}

			command.Arguments = append(command.Arguments, *a)

			// We're done if the type accepts multiple arguments.
			if a.custom != nil || a.manual != nil {
				command.Variadic = true // treat as variadic
				break
			}
		}
	}

	return &command
}

func (cctx *MethodContext) addMiddleware(mw *MiddlewareContext) {
	// Skip if mismatch type:
	if !mw.command.isEvent(cctx.command.event) {
		return
	}
	cctx.middlewares = append(cctx.middlewares, mw)
}

func (cctx *MethodContext) walkMiddlewares(ev reflect.Value) error {
	for _, mw := range cctx.middlewares {
		_, err := mw.call(ev)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cctx *MethodContext) Usage() []string {
	if len(cctx.Arguments) == 0 {
		return nil
	}

	var arguments = make([]string, len(cctx.Arguments))
	for i, arg := range cctx.Arguments {
		arguments[i] = arg.String
	}

	return arguments
}

// SetName sets the command name.
func (cctx *MethodContext) SetName(name string) {
	cctx.Command = name
}

// SetArgumentNames sets all the arguments' names using a single call for
// convenience. It is useful for integration commands. The function panics if
// the method isn't a MessageCreate or InteractionCreate handler.
func (cctx *MethodContext) SetArgumentNames(names ...string) {
	if !cctx.isInteractable() {
		panic("method is not a MessageCreate or InteractionCreate handler.")
	}

	for i := 0; i < len(names) && i < len(cctx.Arguments); i++ {
		cctx.Arguments[i].String = names[i]
	}
}

type MiddlewareContext struct {
	command
}

// ParseMiddleware parses a middleware function. This function panics.
func ParseMiddleware(mw interface{}) *MiddlewareContext {
	value := reflect.ValueOf(mw)
	methodT := value.Type()
	numArgs := methodT.NumIn()

	if numArgs != 1 {
		panic("Invalid argument signature for " + methodT.String())
	}

	// Check number of returns:
	numOut := methodT.NumOut()

	// Returns can either be:
	// Nothing  - func()
	// An error - func() error
	if numOut > 1 {
		panic("Invalid return signature for " + methodT.String())
	}

	// Check the last return's type if the method returns anything.
	if numOut == 1 {
		if i := methodT.Out(0); i == nil || !i.Implements(typeIError) {
			panic("unexpected return type (not error) for " + methodT.String())
		}
	}

	var middleware = MiddlewareContext{
		command: newCommand(value, methodT.In(0)),
	}

	return &middleware
}
