package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

var (
	typeMessageCreate = reflect.TypeOf((*gateway.MessageCreateEvent)(nil))
	// typeof.Implements(typeI*)
	typeValues  = reflect.TypeOf((*Values)(nil)).Elem()
	typeIError  = reflect.TypeOf((*error)(nil)).Elem()
	typeIManP   = reflect.TypeOf((*ManualParseable)(nil)).Elem()
	typeIParser = reflect.TypeOf((*Parseable)(nil)).Elem()
	typeIUsager = reflect.TypeOf((*Usager)(nil)).Elem()
)

type Subcommand struct {
	Description string

	// Commands contains all the registered command contexts.
	Commands []*CommandContext

	// struct name
	name string

	// struct flags
	Flag NameFlag

	// Directly to struct
	cmdValue reflect.Value
	cmdType  reflect.Type

	// Pointer value
	ptrValue reflect.Value
	ptrType  reflect.Type

	// Middleware methods
	mwMethods []*CommandContext

	// command interface as reference
	command interface{}
}

// CommandContext is an internal struct containing fields to make this library
// work. As such, they're all unexported. Description, however, is exported for
// editing, and may be used to generate more informative help messages.
type CommandContext struct {
	Description string
	Flag        NameFlag

	name   string        // all lower-case
	value  reflect.Value // Func
	event  reflect.Type  // gateway.*Event
	method reflect.Method

	// values store
	Values bool
	values reflect.Value

	// equal slices
	argStrings []string
	arguments  []argumentValueFn

	// only for ParseContent interface
	parseMethod reflect.Method
	parseType   reflect.Type
	parseUsage  string
}

// Descriptor is optionally used to set the Description of a command context.
type Descriptor interface {
	Description() string
}

// Namer is optionally used to override the command context's name.
type Namer interface {
	Name() string
}

// Usager is optionally used to override the generated usage for either an
// argument, or multiple (using ManualParseable).
type Usager interface {
	Usage() string
}

func (cctx *CommandContext) Name() string {
	return cctx.name
}

func (cctx *CommandContext) Usage() []string {
	if cctx.parseType != nil {
		return []string{cctx.parseUsage}
	}

	if len(cctx.arguments) == 0 {
		return nil
	}

	return cctx.argStrings
}

func NewSubcommand(cmd interface{}) (*Subcommand, error) {
	var sub = Subcommand{
		command: cmd,
	}

	// Set description
	if d, ok := cmd.(Descriptor); ok {
		sub.Description = d.Description()
	}

	if err := sub.reflectCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to reflect commands")
	}

	if err := sub.parseCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to parse commands")
	}

	return &sub, nil
}

// Name returns the command name in lower case. This only returns non-zero for
// subcommands.
func (sub *Subcommand) Name() string {
	return sub.name
}

// NeedsName sets the name for this subcommand. Like InitCommands, this
// shouldn't be called at all, rather you should use RegisterSubcommand.
func (sub *Subcommand) NeedsName() {
	flag, name := ParseFlag(sub.cmdType.Name())

	// Check for interface
	if n, ok := sub.command.(Namer); ok {
		name = n.Name()
	}

	if !flag.Is(Raw) {
		name = strings.ToLower(name)
	}

	sub.name = name
	sub.Flag = flag
}

func (sub *Subcommand) reflectCommands() error {
	t := reflect.TypeOf(sub.command)
	v := reflect.ValueOf(sub.command)

	if t.Kind() != reflect.Ptr {
		return errors.New("sub is not a pointer")
	}

	// Set the pointer fields
	sub.ptrValue = v
	sub.ptrType = t

	ts := t.Elem()
	vs := v.Elem()

	if ts.Kind() != reflect.Struct {
		return errors.New("sub is not pointer to struct")
	}

	// Set the struct fields
	sub.cmdValue = vs
	sub.cmdType = ts

	return nil
}

// InitCommands fills a Subcommand with a context. This shouldn't be called at
// all, rather you should use the RegisterSubcommand method of a Context.
func (sub *Subcommand) InitCommands(ctx *Context) error {
	// Start filling up a *Context field
	for i := 0; i < sub.cmdValue.NumField(); i++ {
		field := sub.cmdValue.Field(i)

		if !field.CanSet() || !field.CanInterface() {
			continue
		}

		if _, ok := field.Interface().(*Context); !ok {
			continue
		}

		field.Set(reflect.ValueOf(ctx))
		return nil
	}

	return errors.New("No fields with *Command found")
}

func (sub *Subcommand) parseCommands() error {
	var numMethods = sub.ptrValue.NumMethod()
	var commands = make([]*CommandContext, 0, numMethods)

	for i := 0; i < numMethods; i++ {
		method := sub.ptrValue.Method(i)

		if !method.CanInterface() {
			continue
		}

		methodT := method.Type()
		numArgs := methodT.NumIn()

		// Doesn't meet requirement for an event
		if numArgs == 0 {
			continue
		}

		// Check return type
		if err := methodT.Out(0); err == nil || !err.Implements(typeIError) {
			// Invalid, skip
			continue
		}

		var command = CommandContext{
			method: sub.ptrType.Method(i),
			value:  method,
			event:  methodT.In(0), // parse event
		}

		// Parse the method name
		flag, name := ParseFlag(command.method.Name)

		if !flag.Is(Raw) {
			name = strings.ToLower(name)
		}

		// Set the method name and flag
		command.name = name
		command.Flag = flag

		// The argument to start iterating over
		var start = 1

		// TODO: allow more flexibility
		if command.event != typeMessageCreate {
			goto Done
		}

		// If the method only takes an event:
		if numArgs == 1 {
			// done
			goto Done
		}

		// If the method's second argument is Values:
		if t := methodT.In(start); t == typeValues {
			command.Values = true

			// Increment arguments index by 1
			start++
		}

		// middlewares shouldn't even have arguments
		if flag.Is(Middleware) {
			goto Done
		}

		// If the second argument implements ParseContent()
		if t := methodT.In(start); t.Implements(typeIManP) {
			mt, _ := t.MethodByName("ParseContent")

			command.parseMethod = mt
			command.parseType = t.Elem()

			command.parseUsage = usager(t)
			if command.parseUsage == "" {
				command.parseUsage = t.String()
			}

			goto Done
		}

		command.arguments = make([]argumentValueFn, 0, numArgs)

		// Fill up arguments
		for i := start; i < numArgs; i++ {
			t := methodT.In(i)

			avfs, err := getArgumentValueFn(t)
			if err != nil {
				return errors.Wrap(err, "Error parsing argument "+t.String())
			}

			command.arguments = append(command.arguments, avfs)

			var usage = usager(t)
			if usage == "" {
				usage = t.String()
			}

			command.argStrings = append(command.argStrings, usage)
		}

	Done:
		// Append
		if flag.Is(Middleware) {
			sub.mwMethods = append(sub.mwMethods, &command)
		} else {
			commands = append(commands, &command)
		}
	}

	sub.Commands = commands
	return nil
}

func usager(t reflect.Type) string {
	if !t.Implements(typeIUsager) {
		return ""
	}

	usageFn, _ := t.MethodByName("Usage")
	v := usageFn.Func.Call([]reflect.Value{
		reflect.New(t.Elem()),
	})
	return v[0].String()
}
