package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

var (
	typeMessageCreate = reflect.TypeOf((*gateway.MessageCreateEvent)(nil))

	typeString = reflect.TypeOf("")
	typeEmbed  = reflect.TypeOf((*discord.Embed)(nil))
	typeSend   = reflect.TypeOf((*api.SendMessageData)(nil))

	typeSubcmd = reflect.TypeOf((*Subcommand)(nil))

	typeIError  = reflect.TypeOf((*error)(nil)).Elem()
	typeIManP   = reflect.TypeOf((*ManualParser)(nil)).Elem()
	typeICusP   = reflect.TypeOf((*CustomParser)(nil)).Elem()
	typeIParser = reflect.TypeOf((*Parser)(nil)).Elem()
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

	// Commands can actually return either a string, an embed, or a
	// SendMessageData, with error as the second argument.

	// All registered command contexts:
	Commands []*CommandContext
	Events   []*CommandContext

	// Middleware command contexts:
	mwMethods []*CommandContext

	// struct flags
	Flag NameFlag

	// SanitizeMessage is executed on the message content if the method returns
	// a string content or a SendMessageData.
	SanitizeMessage func(content string) string

	// Plumb nameflag, use Commands[0] if true.
	plumb bool

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
	Command    string // empty if Plumb

	value  reflect.Value // Func
	event  reflect.Type  // gateway.*Event
	method reflect.Method

	// return type
	retType reflect.Type

	Arguments []Argument
}

// CanSetup is used for subcommands to change variables, such as Description.
// This method will be triggered when InitCommands is called, which is during
// New for Context and during RegisterSubcommand for subcommands.
type CanSetup interface {
	// Setup should panic when it has an error.
	Setup(*Subcommand)
}

func (cctx *CommandContext) Usage() []string {
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
		SanitizeMessage: func(c string) string {
			return c
		},
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
		name = lowerFirstLetter(name)
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

func (sub *Subcommand) Help(prefix, indent string, hideAdmin bool) string {
	if sub.Flag.Is(AdminOnly) && hideAdmin {
		return ""
	}

	// The header part:
	var header string

	if sub.Command != "" {
		header += indent + sub.Command
	}

	if sub.Description != "" {
		if header != "" {
			header += ": "
		} else {
			header += indent
		}

		header += sub.Description
	}

	header += "\n"

	// The commands part:
	var commands = ""

	for _, cmd := range sub.Commands {
		if cmd.Flag.Is(AdminOnly) && hideAdmin {
			continue
		}

		commands += indent + indent +
			prefix + sub.Command + " " + cmd.Command

		switch {
		case len(cmd.Usage()) > 0:
			commands += " " + strings.Join(cmd.Usage(), " ")
		case cmd.Description != "":
			commands += ": " + cmd.Description
		}

		commands += "\n"
	}

	if commands == "" {
		return ""
	}

	return header + commands
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

	return errors.New("No fields with *bot.Context found")
}

func (sub *Subcommand) parseCommands() error {
	var numMethods = sub.ptrValue.NumMethod()

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
		numOut := methodT.NumOut()
		if numOut == 0 || numOut > 2 {
			continue
		}

		// Check the last return's type:
		if i := methodT.Out(numOut - 1); i == nil || !i.Implements(typeIError) {
			// Invalid, skip.
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
			command.Command = lowerFirstLetter(name)
		}

		// Middlewares shouldn't even have arguments.
		if flag.Is(Middleware) {
			sub.mwMethods = append(sub.mwMethods, &command)
			continue
		}

		// TODO: allow more flexibility
		if command.event != typeMessageCreate || flag.Is(Hidden) {
			sub.Events = append(sub.Events, &command)
			continue
		}

		// See if we know the first return type, if error's return is the
		// second:
		if numOut > 1 {
			switch t := methodT.Out(0); t {
			case typeString, typeEmbed, typeSend:
				// noop, passes
			default:
				continue
			}
		}

		// If a plumb method has been found:
		if sub.plumb {
			continue
		}

		// If the method only takes an event:
		if numArgs == 1 {
			sub.Commands = append(sub.Commands, &command)
			continue
		}

		// The argument's second argument (the first is the event).
		var inT = methodT.In(1)
		var ptr bool

		if inT.Kind() != reflect.Ptr {
			inT = reflect.PtrTo(inT)
			ptr = true
		}

		// If the second argument implements CustomParse()
		if t := inT; t.Implements(typeICusP) {
			mt, _ := inT.MethodByName("CustomParse")

			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}

			command.Arguments = []Argument{{
				String:  t.String(),
				Type:    t,
				pointer: ptr,
				custom:  &mt,
			}}

			goto Done
		}

		// If the second argument implements ParseContent()
		if t := inT; t.Implements(typeIManP) {
			mt, _ := inT.MethodByName("ParseContent")

			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}

			command.Arguments = []Argument{{
				String:  t.String(),
				Type:    t,
				pointer: ptr,
				manual:  &mt,
			}}

			goto Done
		}

		command.Arguments = make([]Argument, 0, numArgs)

		// Fill up arguments
		for i := 1; i < numArgs; i++ {
			t := methodT.In(i)
			a, err := getArgumentValueFn(t)
			if err != nil {
				return errors.Wrap(err, "Error parsing argument "+t.String())
			}

			command.Arguments = append(command.Arguments, *a)
		}

	Done:
		// If the current event is a plumb event:
		if flag.Is(Plumb) {
			command.Command = "" // plumbers don't have names
			sub.Commands = []*CommandContext{&command}
			sub.plumb = true
			continue
		}

		// Append
		sub.Commands = append(sub.Commands, &command)
	}

	return nil
}

func lowerFirstLetter(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}
