package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

var (
	typeMessageCreate = reflect.TypeOf((*gateway.MessageCreateEvent)(nil))

	typeSubcmd = reflect.TypeOf((*Subcommand)(nil))

	typeIError  = reflect.TypeOf((*error)(nil)).Elem()
	typeIManP   = reflect.TypeOf((*ManualParseable)(nil)).Elem()
	typeIParser = reflect.TypeOf((*Parseable)(nil)).Elem()
	typeSetupFn = func() reflect.Type {
		method, _ := reflect.TypeOf((*CanSetup)(nil)).
			Elem().
			MethodByName("Setup")
		return method.Type
	}()
)

type Subcommand struct {
	Description string

	// Raw struct name, including the flag (only filled for actual subcommands,
	// will be empty for Context):
	StructName string
	// Parsed command name:
	Command string

	// All registered command contexts:
	Commands []*CommandContext

	// Middleware command contexts:
	mwMethods []*CommandContext

	// struct flags
	Flag NameFlag

	// Directly to struct
	cmdValue reflect.Value
	cmdType  reflect.Type

	// Pointer value
	ptrValue reflect.Value
	ptrType  reflect.Type

	// command interface as reference
	command interface{}
}

// CommandContext is an internal struct containing fields to make this library
// work. As such, they're all unexported. Description, however, is exported for
// editing, and may be used to generate more informative help messages.
type CommandContext struct {
	Description string
	Flag        NameFlag

	MethodName string
	Command    string

	value  reflect.Value // Func
	event  reflect.Type  // gateway.*Event
	method reflect.Method

	Arguments []Argument

	// only for ParseContent interface
	parseMethod reflect.Method
	parseType   reflect.Type
	parseUsage  string
}

// CanSetup is used for subcommands to change variables, such as Description.
// This method will be triggered when InitCommands is called, which is during
// New for Context and during RegisterSubcommand for subcommands.
type CanSetup interface {
	// Setup should panic when it has an error.
	Setup(*Subcommand)
}

func (cctx *CommandContext) Usage() []string {
	if cctx.parseType != nil {
		return []string{cctx.parseUsage}
	}

	if len(cctx.Arguments) == 0 {
		return nil
	}

	var arguments = make([]string, len(cctx.Arguments))
	for i, arg := range cctx.Arguments {
		arguments[i] = arg.String
	}

	return arguments
}

func NewSubcommand(cmd interface{}) (*Subcommand, error) {
	var sub = Subcommand{
		command: cmd,
	}

	if err := sub.reflectCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to reflect commands")
	}

	if err := sub.parseCommands(); err != nil {
		return nil, errors.Wrap(err, "Failed to parse commands")
	}

	return &sub, nil
}

// NeedsName sets the name for this subcommand. Like InitCommands, this
// shouldn't be called at all, rather you should use RegisterSubcommand.
func (sub *Subcommand) NeedsName() {
	sub.StructName = sub.cmdType.Name()

	flag, name := ParseFlag(sub.StructName)

	if !flag.Is(Raw) {
		name = strings.ToLower(name)
	}

	sub.Command = name
	sub.Flag = flag
}

// ChangeCommandInfo changes the matched methodName's Command and Description.
// Empty means unchanged. The returned bool is true when the method is found.
func (sub *Subcommand) ChangeCommandInfo(methodName, cmd, desc string) bool {
	for _, c := range sub.Commands {
		if c.MethodName != methodName {
			continue
		}

		if cmd != "" {
			c.Command = cmd
		}
		if desc != "" {
			c.Description = desc
		}

		return true
	}

	return false
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
	if err := sub.fillStruct(ctx); err != nil {
		return err
	}

	// See if struct implements CanSetup:
	if v, ok := sub.command.(CanSetup); ok {
		v.Setup(sub)
	}

	return nil
}

func (sub *Subcommand) fillStruct(ctx *Context) error {
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

		if numArgs == 0 {
			// Doesn't meet the requirement for an event, continue.
			continue
		}

		if methodT == typeSetupFn {
			// Method is a setup method, continue.
			continue
		}

		// Check number of returns:
		if methodT.NumOut() != 1 {
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

		// Set the method name, command, and flag:
		command.MethodName = name
		command.Command = name
		command.Flag = flag

		// Check if Raw is enabled for command:
		if !flag.Is(Raw) {
			command.Command = strings.ToLower(name)
		}

		// TODO: allow more flexibility
		if command.event != typeMessageCreate {
			goto Done
		}

		// If the method only takes an event:
		if numArgs == 1 {
			// done
			goto Done
		}

		// Middlewares shouldn't even have arguments.
		if flag.Is(Middleware) {
			goto Done
		}

		// If the second argument implements ParseContent()
		if t := methodT.In(1); t.Implements(typeIManP) {
			mt, _ := t.MethodByName("ParseContent")

			command.parseMethod = mt
			command.parseType = t.Elem()
			command.parseUsage = t.String()

			goto Done
		}

		command.Arguments = make([]Argument, 0, numArgs)

		// Fill up arguments
		for i := 1; i < numArgs; i++ {
			t := methodT.In(i)

			avfs, err := getArgumentValueFn(t)
			if err != nil {
				return errors.Wrap(err, "Error parsing argument "+t.String())
			}

			command.Arguments = append(command.Arguments, Argument{
				String: t.String(),
				Type:   t,
				fn:     avfs,
			})
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
