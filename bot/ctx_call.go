package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

// NonFatal is an interface that a method can implement to ignore all errors.
// This works similarly to Break.
type NonFatal interface {
	error
	IgnoreError() // noop method
}

func onlyFatal(err error) error {
	if _, ok := err.(NonFatal); ok {
		return nil
	}
	return err
}

type _Break struct{ error }

// implement NonFatal.
func (_Break) IgnoreError() {}

// Break is a non-fatal error that could be returned from middlewares or
// handlers to stop the chain of execution.
//
// Middlewares are guaranteed to be executed before handlers, but the exact
// order of each are undefined. Main handlers are also guaranteed to be executed
// before all subcommands. If a main middleware cancels, no subcommand
// middlewares will be called.
//
// Break implements the NonFatal interface, which causes an error to be ignored.
var Break NonFatal = _Break{errors.New("break middleware chain, non-fatal")}

func (ctx *Context) filterEventType(evT reflect.Type) []*CommandContext {
	var callers []*CommandContext
	var middles []*CommandContext
	var found bool

	find := func(sub *Subcommand) {
		for _, cmd := range sub.Events {
			// Search only for callers, so skip middlewares.
			if cmd.Flag.Is(Middleware) {
				continue
			}

			if cmd.event == evT {
				callers = append(callers, cmd)
				found = true
			}
		}

		// Only get middlewares if we found handlers for that same event.
		if found {
			// Search for middlewares with the same type:
			for _, mw := range sub.mwMethods {
				if mw.event == evT {
					middles = append(middles, mw)
				}
			}
		}
	}

	// Find the main context first.
	find(ctx.Subcommand)

	for _, sub := range ctx.subcommands {
		// Reset found status
		found = false
		// Find subcommands second.
		find(sub)
	}

	return append(middles, callers...)
}

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	var isAdmin *bool // I want to die.
	var isGuild *bool
	var callers []*CommandContext

	// Hit the cache
	t, ok := ctx.typeCache.Load(evT)
	if ok {
		callers = t.([]*CommandContext)
	} else {
		callers = ctx.filterEventType(evT)
		ctx.typeCache.Store(evT, callers)
	}

	// We can't do the callers[:0] trick here, as it will modify the slice
	// inside the sync.Map as well.
	var filtered = make([]*CommandContext, 0, len(callers))

	for _, cmd := range callers {
		// Command flags will inherit its parent Subcommand's flags.
		if true &&
			!(cmd.Flag.Is(AdminOnly) && !ctx.eventIsAdmin(ev, &isAdmin)) &&
			!(cmd.Flag.Is(GuildOnly) && !ctx.eventIsGuild(ev, &isGuild)) {

			filtered = append(filtered, cmd)
		}
	}

	for _, c := range filtered {
		_, err := callWith(c.value, ev)
		if err != nil {
			if err = onlyFatal(err); err != nil {
				ctx.ErrorLogger(err)
			}
			return err
		}
	}

	// We call the messages later, since Hidden handlers will go into the Events
	// slice, but we don't want to ignore those handlers either.
	if evT == typeMessageCreate {
		// safe assertion always
		err := ctx.callMessageCreate(ev.(*gateway.MessageCreateEvent))
		return onlyFatal(err)
	}

	return nil
}

func (ctx *Context) callMessageCreate(mc *gateway.MessageCreateEvent) error {
	// check if prefix
	pf, ok := ctx.HasPrefix(mc)
	if !ok {
		return nil
	}

	// trim the prefix before splitting, this way multi-words prefices work
	content := mc.Content[len(pf):]

	if content == "" {
		return nil // just the prefix only
	}

	// parse arguments
	args, err := ParseArgs(content)
	if err != nil {
		return errors.Wrap(err, "Failed to parse command")
	}

	if len(args) == 0 {
		return nil // ???
	}

	var cmd *CommandContext
	var sub *Subcommand
	var start int // arg starts from $start

	// Check if plumb:
	if ctx.plumb {
		cmd = ctx.Commands[0]
		sub = ctx.Subcommand
		start = 0
	}

	// If not plumb, search for the command
	if cmd == nil {
		for _, c := range ctx.Commands {
			if c.Command == args[0] {
				cmd = c
				sub = ctx.Subcommand
				start = 1
				break
			}
		}
	}

	// Can't find the command, look for subcommands if len(args) has a 2nd
	// entry.
	if cmd == nil {
		for _, s := range ctx.subcommands {
			if s.Command != args[0] {
				continue
			}

			// Check if plumb:
			if s.plumb {
				cmd = s.Commands[0]
				sub = s
				start = 1
				break
			}

			// There's no second argument, so we can only look for Plumbed
			// subcommands.
			if len(args) < 2 {
				continue
			}

			for _, c := range s.Commands {
				if c.Command == args[1] {
					cmd = c
					sub = s
					start = 2
				}
			}

			if cmd == nil {
				if s.QuietUnknownCommand {
					return nil
				}

				return &ErrUnknownCommand{
					Command: args[1],
					Parent:  args[0],
					ctx:     s.Commands,
				}
			}

			break
		}
	}

	if cmd == nil {
		if ctx.QuietUnknownCommand {
			return nil
		}

		return &ErrUnknownCommand{
			Command: args[0],
			ctx:     ctx.Commands,
		}
	}

	// Check for IsAdmin and IsGuild
	if cmd.Flag.Is(GuildOnly) && !mc.GuildID.Valid() {
		return nil
	}
	if cmd.Flag.Is(AdminOnly) {
		p, err := ctx.State.Permissions(mc.ChannelID, mc.Author.ID)
		if err != nil || !p.Has(discord.PermissionAdministrator) {
			return nil
		}
	}

	// Start converting
	var argv []reflect.Value

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.Arguments) < 1 {
		goto Call
	}

	// Check manual or parser
	if cmd.Arguments[0].fn == nil {
		// Create a zero value instance of this:
		v := reflect.New(cmd.Arguments[0].Type)
		ret := []reflect.Value{}

		switch {
		case cmd.Arguments[0].manual != nil:
			// Pop out the subcommand name, if there's one:
			if sub.Command != "" {
				args = args[1:]
			}

			// Call the manual parse method:
			ret = cmd.Arguments[0].manual.Func.Call([]reflect.Value{
				v, reflect.ValueOf(args),
			})

		case cmd.Arguments[0].custom != nil:
			// For consistent behavior, clear the subcommand name off:
			content = content[len(sub.Command):]
			// Trim space if there are any:
			content = strings.TrimSpace(content)

			// Call the method with the raw unparsed command:
			ret = cmd.Arguments[0].custom.Func.Call([]reflect.Value{
				v, reflect.ValueOf(content),
			})
		}

		// Check the returned error:
		_, err := errorReturns(ret)
		if err != nil {
			return err
		}

		// Check if the argument wants a non-pointer:
		if cmd.Arguments[0].pointer {
			v = v.Elem()
		}

		// Add the argument to the list of arguments:
		argv = append(argv, v)
		goto Call
	}

	// Not enough arguments given
	if delta := len(args[start:]) - len(cmd.Arguments); delta != 0 {
		var err = "Not enough arguments given"
		if delta > 0 {
			err = "Too many arguments given"
		}

		return &ErrInvalidUsage{
			Args:  args,
			Index: len(args) - 1,
			Err:   err,
			Ctx:   cmd,
		}
	}

	argv = make([]reflect.Value, len(cmd.Arguments))

	for i := start; i < len(args); i++ {
		v, err := cmd.Arguments[i-start].fn(args[i])
		if err != nil {
			return &ErrInvalidUsage{
				Args:  args,
				Index: i,
				Err:   err.Error(),
				Ctx:   cmd,
			}
		}

		argv[i-start] = v
	}

Call:
	// Try calling all middlewares first. We don't need to stack middlewares, as
	// there will only be one command match.
	for _, mw := range sub.mwMethods {
		_, err := callWith(mw.value, mc)
		if err != nil {
			return err
		}
	}

	// call the function and parse the error return value
	v, err := callWith(cmd.value, mc, argv...)
	if err != nil {
		return err
	}

	switch v := v.(type) {
	case string:
		v = sub.SanitizeMessage(v)
		_, err = ctx.SendMessage(mc.ChannelID, v, nil)
	case *discord.Embed:
		_, err = ctx.SendMessage(mc.ChannelID, "", v)
	case *api.SendMessageData:
		if v.Content != "" {
			v.Content = sub.SanitizeMessage(v.Content)
		}
		_, err = ctx.SendMessageComplex(mc.ChannelID, *v)
	}

	return err
}

func (ctx *Context) eventIsAdmin(ev interface{}, is **bool) bool {
	if *is != nil {
		return **is
	}

	var channelID = reflectChannelID(ev)
	if !channelID.Valid() {
		return false
	}

	var userID = reflectUserID(ev)
	if !userID.Valid() {
		return false
	}

	var res bool

	p, err := ctx.State.Permissions(channelID, userID)
	if err == nil && p.Has(discord.PermissionAdministrator) {
		res = true
	}

	*is = &res
	return res
}

func (ctx *Context) eventIsGuild(ev interface{}, is **bool) bool {
	if *is != nil {
		return **is
	}

	var channelID = reflectChannelID(ev)
	if !channelID.Valid() {
		return false
	}

	c, err := ctx.State.Channel(channelID)
	if err != nil {
		return false
	}

	res := c.GuildID.Valid()
	*is = &res
	return res
}

func callWith(
	caller reflect.Value,
	ev interface{}, values ...reflect.Value) (interface{}, error) {

	return errorReturns(caller.Call(append(
		[]reflect.Value{reflect.ValueOf(ev)},
		values...,
	)))
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

func reflectChannelID(_struct interface{}) discord.Snowflake {
	return _reflectID(reflect.ValueOf(_struct), "Channel")
}

func reflectGuildID(_struct interface{}) discord.Snowflake {
	return _reflectID(reflect.ValueOf(_struct), "Guild")
}

func reflectUserID(_struct interface{}) discord.Snowflake {
	return _reflectID(reflect.ValueOf(_struct), "User")
}

func _reflectID(v reflect.Value, thing string) discord.Snowflake {
	if !v.IsValid() {
		return 0
	}

	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()

		// Recheck after dereferring
		if !v.IsValid() {
			return 0
		}

		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return 0
	}

	numFields := t.NumField()

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		fType := field.Type

		if fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
		}

		switch fType.Kind() {
		case reflect.Struct:
			if chID := _reflectID(v.Field(i), thing); chID.Valid() {
				return chID
			}
		case reflect.Int64:
			if field.Name == thing+"ID" {
				// grab value real quick
				return discord.Snowflake(v.Field(i).Int())
			}

			// Special case where the struct name has Channel in it
			if field.Name == "ID" && strings.Contains(t.Name(), thing) {
				return discord.Snowflake(v.Field(i).Int())
			}
		}
	}

	return 0
}
