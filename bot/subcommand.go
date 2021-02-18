package bot

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/gateway"
)

var (
	typeMessageCreate     = reflect.TypeOf((*gateway.MessageCreateEvent)(nil))
	typeMessageUpdate     = reflect.TypeOf((*gateway.MessageUpdateEvent)(nil))
	typeInteractionCreate = reflect.TypeOf((*gateway.InteractionCreateEvent)(nil))

	typeIError  = reflect.TypeOf((*error)(nil)).Elem()
	typeIManP   = reflect.TypeOf((*ManualParser)(nil)).Elem()
	typeICusP   = reflect.TypeOf((*CustomParser)(nil)).Elem()
	typeIParser = reflect.TypeOf((*Parser)(nil)).Elem()
	typeIUsager = reflect.TypeOf((*Usager)(nil)).Elem()
	typeSetupFn = methodType((*CanSetup)(nil), "Setup")
)

func methodType(iface interface{}, name string) reflect.Type {
	method, _ := reflect.TypeOf(iface).
		Elem().
		MethodByName(name)
	return method.Type
}

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
// or events. A return type T implies that return value will be ignored.
//
//    func(*gateway.MessageCreateEvent, ...) (string, error)
//    func(*gateway.MessageCreateEvent, ...) (*discord.Embed, error)
//    func(*gateway.MessageCreateEvent, ...) (*api.SendMessageData, error)
//    func(*gateway.MessageCreateEvent, ...) (T, error)
//    func(*gateway.MessageCreateEvent, ...) error
//    func(*gateway.MessageCreateEvent, ...)
//
//    func(*gateway.InteractionCreateEvent, ...) (string, error)
//    func(*gateway.InteractionCreateEvent, ...) (*discord.Embed, error)
//    func(*gateway.InteractionCreateEvent, ...) (*api.InteractionResponse, error)
//    func(*gateway.InteractionCreateEvent, ...) (T, error)
//    func(*gateway.InteractionCreateEvent, ...) error
//    func(*gateway.InteractionCreateEvent, ...)
//
//    func(<AnyEvent>) (T, error)
//    func(<AnyEvent>) error
//    func(<AnyEvent>)
//
type Subcommand struct {
	// Description is a string that's appended after the subcommand name in
	// (*Context).Help().
	Description string

	// Hidden if true will not be shown by (*Context).Help(). It will
	// also cause unknown command errors to be suppressed.
	Hidden bool

	// Raw struct name, including the flag (only filled for actual subcommands,
	// will be empty for Context):
	StructName string
	// Parsed command name:
	Command string

	// Aliases is alternative way to call this subcommand in Discord.
	Aliases []string

	// SanitizeMessage is currently no longer used automatically.
	// AllowedMentions is used instead.
	//
	// This field is deprecated and will be removed eventually.
	SanitizeMessage func(content string) string

	// Commands can return either a string, a *discord.Embed, or an
	// *api.SendMessageData, with error as the second argument.

	// All registered method contexts:
	Events   []*MethodContext
	Commands []*MethodContext
	plumbed  *MethodContext

	// Global middlewares.
	globalmws []*MiddlewareContext

	// Directly to struct
	cmdValue reflect.Value
	cmdType  reflect.Type

	// Pointer value
	ptrValue reflect.Value
	ptrType  reflect.Type

	helper  func() string
	command interface{}
}

// CanSetup is used for subcommands to change variables, such as Description.
// This method will be triggered when InitCommands is called, which is during
// New for Context and during RegisterSubcommand for subcommands.
type CanSetup interface {
	// Setup should panic when it has an error.
	Setup(*Subcommand)
}

// CanHelp is an interface that subcommands can implement to return its own help
// message. Those messages will automatically be indented into suitable sections
// by the default Help() implementation. Unlike Usager or CanSetup, the Help()
// method will be called every time it's needed.
type CanHelp interface {
	Help() string
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
		return nil, errors.Wrap(err, "failed to reflect commands")
	}

	if err := sub.parseCommands(); err != nil {
		return nil, errors.Wrap(err, "failed to parse commands")
	}

	return &sub, nil
}

// NeedsName sets the name for this subcommand. Like InitCommands, this
// shouldn't be called at all, rather you should use RegisterSubcommand.
func (sub *Subcommand) NeedsName() {
	sub.StructName = sub.cmdType.Name()
	sub.Command = lowerFirstLetter(sub.StructName)
}

func lowerFirstLetter(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}

// FindCommand finds the MethodContext using either the given method or the
// given method name. It panics if the given method is not found.
//
// There are two ways to use FindCommand:
//
//    sub.FindCommand("MethodName")
//    sub.FindCommand(thing.MethodName)
//
func (sub *Subcommand) FindCommand(method interface{}) *MethodContext {
	return sub.findMethod(method, false)
}

func (sub *Subcommand) findMethod(method interface{}, inclEvents bool) *MethodContext {
	methodName, ok := method.(string)
	if !ok {
		methodName = runtimeMethodName(method)
	}

	for _, c := range sub.Commands {
		if c.MethodName == methodName {
			return c
		}
	}

	if inclEvents {
		for _, ev := range sub.Events {
			if ev.MethodName == methodName {
				return ev
			}
		}
	}

	panic("can't find method " + methodName)
}

// runtimeMethodName returns the name of the method from the given method call.
// It is used as such:
//
//    fmt.Println(methodName(t.Method_dash))
//    // Output: main.T.Method_dash-fm
//
func runtimeMethodName(v interface{}) string {
	// https://github.com/diamondburned/arikawa/issues/146

	ptr := reflect.ValueOf(v).Pointer()

	funcPC := runtime.FuncForPC(ptr)
	if funcPC == nil {
		panic("given method is not a function")
	}

	funcName := funcPC.Name()

	// Do weird string parsing because Go wants us to.
	nameParts := strings.Split(funcName, ".")
	mName := nameParts[len(nameParts)-1]
	nameParts = strings.Split(mName, "-")
	if len(nameParts) > 1 { // extract the string before -fm if possible
		mName = nameParts[len(nameParts)-2]
	}

	return mName
}

// ChangeCommandInfo changes the matched method's Command and Description.
// Empty means unchanged. This function panics if the given method is not found.
func (sub *Subcommand) ChangeCommandInfo(method interface{}, cmd, desc string) {
	var command = sub.FindCommand(method)
	if cmd != "" {
		command.Command = cmd
	}
	if desc != "" {
		command.Description = desc
	}
}

// SetArgumentNames is a convenient wrapper for MethodContext's
// SetArgumentNames. It is useful for integration commands.
func (sub *Subcommand) SetArgumentNames(method interface{}, names ...string) {
	sub.FindCommand(method).SetArgumentNames(names...)
}

// Help calls the subcommand's Help() or auto-generates one with HelpGenerate()
// if the subcommand doesn't implement CanHelp. It doesn't show hidden commands
// by default.
func (sub *Subcommand) Help() string {
	return sub.HelpShowHidden(false)
}

// HelpShowHidden does the same as Help(), except it will render hidden commands
// if the subcommand doesn't implement CanHelp and showHiddeen is true.
func (sub *Subcommand) HelpShowHidden(showHidden bool) string {
	// Check if the subcommand implements CanHelp.
	if sub.helper != nil {
		return sub.helper()
	}
	return sub.HelpGenerate(showHidden)
}

// HelpGenerate auto-generates a help message, which contains only a list of
// commands. It does not print the subcommand header. Use this only if you want
// to override the Subcommand's help, else use Help(). This function will show
// hidden commands if showHidden is true.
func (sub *Subcommand) HelpGenerate(showHidden bool) string {
	var buf strings.Builder

	for i, cmd := range sub.Commands {
		if cmd.Hidden && !showHidden {
			continue
		}

		if sub.Command != "" {
			buf.WriteString(sub.Command)
			buf.WriteByte(' ')
		}

		if cmd == sub.PlumbedMethod() {
			buf.WriteByte('[')
		}

		buf.WriteString(cmd.Command)

		for _, alias := range cmd.Aliases {
			buf.WriteByte('|')
			buf.WriteString(alias)
		}

		if cmd == sub.PlumbedMethod() {
			buf.WriteByte(']')
		}

		// Write the usages first.
		var usages = cmd.Usage()

		for _, usage := range usages {
			buf.WriteByte(' ')
			buf.WriteString("__")
			buf.WriteString(usage)
			buf.WriteString("__")
		}

		// Is the last argument trailing? If so, append ellipsis.
		if len(usages) > 0 && cmd.Variadic {
			buf.WriteString("...")
		}

		// Write the description if there's any.
		if cmd.Description != "" {
			buf.WriteString(": ")
			buf.WriteString(cmd.Description)
		}

		// Add a new line if this isn't the last command.
		if i != len(sub.Commands)-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

// Hide marks a command as hidden, meaning it won't be shown in help and its
// UnknownCommand errors will be suppressed.
func (sub *Subcommand) Hide(method interface{}) {
	sub.FindCommand(method).Hidden = true
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

	// See if struct implements CanHelper:
	if v, ok := sub.command.(CanHelp); ok {
		sub.helper = v.Help
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

	return errors.New("no fields with *bot.Context found")
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

		switch cctx.event {
		case typeMessageCreate:
			sub.Commands = append(sub.Commands, cctx)
		case typeInteractionCreate:
			panic("TODO")
		default:
			sub.Events = append(sub.Events, cctx)
		}
	}

	return nil
}

// AddMiddleware adds a middleware into multiple or all methods, including
// commands and events. Multiple method names can be comma-delimited. For all
// methods, use a star (*). The given middleware argument can either be a
// function with one of the allowed methods or a *MiddlewareContext.
//
// Allowed function signatures
//
// Below are the acceptable function signatures that would be parsed as a proper
// middleware. A return value of type T will be ignored. If the given function
// is invalid, then this method will panic.
//
//    func(<AnyEvent>) (T, error)
//    func(<AnyEvent>) error
//    func(<AnyEvent>)
//
// Note that although technically all of the above function signatures are
// acceptable, one should almost always return only an error.
func (sub *Subcommand) AddMiddleware(method, middleware interface{}) {
	var mw *MiddlewareContext
	// Allow *MiddlewareContext to be passed into.
	if v, ok := middleware.(*MiddlewareContext); ok {
		mw = v
	} else {
		mw = ParseMiddleware(middleware)
	}

	switch v := method.(type) {
	case string:
		sub.addMiddleware(mw, strings.Split(v, ","))
	case []string:
		sub.addMiddleware(mw, v)
	default:
		sub.findMethod(v, true).addMiddleware(mw)
	}
}

func (sub *Subcommand) addMiddleware(mw *MiddlewareContext, methods []string) {
	for _, method := range methods {
		// Trim space.
		if method = strings.TrimSpace(method); method == "*" {
			// Append middleware to global middleware slice.
			sub.globalmws = append(sub.globalmws, mw)
			continue
		}
		// Append middleware to that individual function.
		sub.findMethod(method, true).addMiddleware(mw)
	}
}

func (sub *Subcommand) eventCallers(evT reflect.Type) (callers []caller) {
	// Search for global middlewares.
	for _, mw := range sub.globalmws {
		if mw.isEvent(evT) {
			callers = append(callers, mw)
		}
	}

	// Search for specific handlers.
	for _, cctx := range sub.Events {
		// We only take middlewares and callers if the event matches and is not
		// a MessageCreate. The other function already handles that.
		if cctx.isEvent(evT) {
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

// IsPlumbed returns true if the subcommand is plumbed. To get the plumbed
// method, use PlumbedMethod().
func (sub *Subcommand) IsPlumbed() bool {
	return sub.plumbed != nil
}

// PlumbedMethod returns the plumbed method's context, or nil if the subcommand
// is not plumbed.
func (sub *Subcommand) PlumbedMethod() *MethodContext {
	return sub.plumbed
}

// SetPlumb sets the method as the plumbed command. If method is nil, then the
// plumbing is also disabled.
func (sub *Subcommand) SetPlumb(method interface{}) {
	// Ensure that SetPlumb isn't being called on the main context.
	if sub.Command == "" {
		panic("invalid SetPlumb call on *Context")
	}

	if method == nil {
		sub.plumbed = nil
		return
	}

	sub.plumbed = sub.FindCommand(method)
}

// AddAliases add alias(es) to specific command (defined with commandName).
func (sub *Subcommand) AddAliases(commandName interface{}, aliases ...string) {
	// Get command
	command := sub.FindCommand(commandName)

	// Write new listing of aliases
	command.Aliases = append(command.Aliases, aliases...)
}

// DeriveIntents derives all possible gateway intents from the method handlers
// and middlewares.
func (sub *Subcommand) DeriveIntents() gateway.Intents {
	var intents gateway.Intents

	for _, event := range sub.Events {
		intents |= event.intents()
	}
	for _, command := range sub.Commands {
		intents |= command.intents()
	}
	if sub.IsPlumbed() {
		intents |= sub.plumbed.intents()
	}
	for _, middleware := range sub.globalmws {
		intents |= middleware.intents()
	}

	return intents
}
