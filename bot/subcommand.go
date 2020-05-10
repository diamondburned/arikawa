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
	typeIUsager = reflect.TypeOf((*Usager)(nil)).Elem()
	typeSetupFn = func() reflect.Type {
		method, _ := reflect.TypeOf((*CanSetup)(nil)).
			Elem().
			MethodByName("Setup")
		return method.Type
	}()
)

// HelpUnderline formats command arguments with an underline, similar to
// manpages.
var HelpUnderline = true

func underline(word string) string {
	if HelpUnderline {
		return "__" + word + "__"
	}
	return word
}

// Subcommand is any form of command, which could be a top-level command or a
// subcommand.
//
// Allowed method signatures
//
// These are the acceptable function signatures that would be parsed as commands
// or events. A return type <T> implies that return value will be ignored.
//
//    func(*gateway.MessageCreateEvent, ...) (string, error)
//    func(*gateway.MessageCreateEvent, ...) (*discord.Embed, error)
//    func(*gateway.MessageCreateEvent, ...) (*api.SendMessageData, error)
//    func(*gateway.MessageCreateEvent, ...) (T, error)
//    func(*gateway.MessageCreateEvent, ...) error
//    func(*gateway.MessageCreateEvent, ...)
//    func(<AnyEvent>) (T, error)
//    func(<AnyEvent>) error
//    func(<AnyEvent>)
//
type Subcommand struct {
	Description string

	// Raw struct name, including the flag (only filled for actual subcommands,
	// will be empty for Context):
	StructName string
	// Parsed command name:
	Command string

	// SanitizeMessage is executed on the message content if the method returns
	// a string content or a SendMessageData.
	SanitizeMessage func(content string) string

	// QuietUnknownCommand, if true, will not make the bot reply with an unknown
	// command error into the chat. If this is set in Context, it will apply to
	// all other subcommands.
	QuietUnknownCommand bool

	// Commands can actually return either a string, an embed, or a
	// SendMessageData, with error as the second argument.

	// All registered method contexts, including commands:
	Methods []*MethodContext
	plumbed *MethodContext

	// Global middlewares.
	globalmws []*MiddlewareContext

	// Directly to struct
	cmdValue reflect.Value
	cmdType  reflect.Type

	// Pointer value
	ptrValue reflect.Value
	ptrType  reflect.Type

	command interface{}
}

// CanSetup is used for subcommands to change variables, such as Description.
// This method will be triggered when InitCommands is called, which is during
// New for Context and during RegisterSubcommand for subcommands.
type CanSetup interface {
	// Setup should panic when it has an error.
	Setup(*Subcommand)
}

// NewSubcommand is used to make a new subcommand. You usually wouldn't call
// this function, but instead use (*Context).RegisterSubcommand().
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
	sub.Command = lowerFirstLetter(sub.StructName)
}

// FindMethod finds the MethodContext. It panics if methodName is not found.
func (sub *Subcommand) FindMethod(methodName string) *MethodContext {
	for _, c := range sub.Methods {
		if c.MethodName == methodName {
			return c
		}
	}
	panic("Can't find method " + methodName)
}

// ChangeCommandInfo changes the matched methodName's Command and Description.
// Empty means unchanged. The returned bool is true when the command is found.
func (sub *Subcommand) ChangeCommandInfo(methodName, cmd, desc string) bool {
	for _, c := range sub.Methods {
		if c.MethodName != methodName || !c.isEvent(typeMessageCreate) {
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

func (sub *Subcommand) Help(indent string, hideAdmin bool) string {
	// // The header part:
	// var header string

	// if sub.Command != "" {
	// 	header += "**" + sub.Command + "**"
	// }

	// if sub.Description != "" {
	// 	if header != "" {
	// 		header += ": "
	// 	}

	// 	header += sub.Description
	// }

	// header += "\n"

	// // The commands part:
	// var commands = ""

	// for i, cmd := range sub.Commands {
	// 	if cmd.Flag.Is(AdminOnly) && hideAdmin {
	// 		continue
	// 	}

	// 	switch {
	// 	case sub.Command != "" && cmd.Command != "":
	// 		commands += indent + sub.Command + " " + cmd.Command
	// 	case sub.Command != "":
	// 		commands += indent + sub.Command
	// 	default:
	// 		commands += indent + cmd.Command
	// 	}

	// 	// Write the usages first.
	// 	for _, usage := range cmd.Usage() {
	// 		commands += " " + underline(usage)
	// 	}

	// 	// Is the last argument trailing? If so, append ellipsis.
	// 	if cmd.Variadic {
	// 		commands += "..."
	// 	}

	// 	// Write the description if there's any.
	// 	if cmd.Description != "" {
	// 		commands += ": " + cmd.Description
	// 	}

	// 	// Add a new line if this isn't the last command.
	// 	if i != len(sub.Commands)-1 {
	// 		commands += "\n"
	// 	}
	// }

	// if commands == "" {
	// 	return ""
	// }

	// return header + commands

	// TODO
	// TODO: Interface Helper implements Help() string
	return "TODO"
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

		methodT := sub.ptrType.Method(i)
		if methodT.Name == "Setup" && methodT.Type == typeSetupFn {
			continue
		}

		cctx := parseMethod(method, methodT)
		if cctx == nil {
			continue
		}

		// Append.
		sub.Methods = append(sub.Methods, cctx)
	}

	return nil
}

func (sub *Subcommand) AddMiddleware(methodName string, middleware interface{}) {
	var mw *MiddlewareContext
	// Allow *MiddlewareContext to be passed into.
	if v, ok := middleware.(*MiddlewareContext); ok {
		mw = v
	} else {
		mw = ParseMiddleware(middleware)
	}

	// Parse method name:
	for _, method := range strings.Split(methodName, ",") {
		// Trim space.
		if method = strings.TrimSpace(method); method == "*" {
			// Append middleware to global middleware slice.
			sub.globalmws = append(sub.globalmws, mw)
		} else {
			// Append middleware to that individual function.
			sub.FindMethod(method).addMiddleware(mw)
		}
	}
}

func (sub *Subcommand) walkMiddlewares(ev reflect.Value) error {
	for _, mw := range sub.globalmws {
		_, err := mw.call(ev)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sub *Subcommand) eventCallers(evT reflect.Type) (callers []caller) {
	// Search for global middlewares.
	for _, mw := range sub.globalmws {
		if mw.isEvent(evT) {
			callers = append(callers, mw)
		}
	}

	// Search for specific handlers.
	for _, cctx := range sub.Methods {
		// We only take middlewares and callers if the event matches and is not
		// a MessageCreate. The other function already handles that.
		if cctx.event != typeMessageCreate && cctx.isEvent(evT) {
			// Add the command's middlewares first.
			for _, mw := range cctx.middlewares {
				// Concrete struct to interface conversion done implicitly.
				callers = append(callers, mw)
			}

			callers = append(callers, cctx)
		}
	}
	return
}

// SetPlumb sets the method as the plumbed command. This means that all calls
// without the second command argument will call this method in a subcommand. It
// panics if sub.Command is empty.
func (sub *Subcommand) SetPlumb(methodName string) {
	if sub.Command == "" {
		panic("SetPlumb called on a main command with sub.Command empty.")
	}

	method := sub.FindMethod(methodName)
	method.Command = ""
	sub.plumbed = method
}

func lowerFirstLetter(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}
