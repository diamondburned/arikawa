package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/bot/extras/infer"
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
	// check if bot
	if !ctx.AllowBot && mc.Author.Bot {
		return nil
	}

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
	parts, err := ctx.ParseArgs(content)
	if err != nil {
		return errors.Wrap(err, "Failed to parse command")
	}

	if len(parts) == 0 {
		return nil // ???
	}

	var cmd *CommandContext
	var sub *Subcommand
	// var start int // arg starts from $start

	// Check if plumb:
	if ctx.plumb {
		cmd = ctx.Commands[0]
		sub = ctx.Subcommand
		// start = 0
	}

	// Arguments slice, which will be sliced away until only arguments are left.
	var arguments = parts

	// If not plumb, search for the command
	if cmd == nil {
		for _, c := range ctx.Commands {
			if c.Command == parts[0] {
				cmd = c
				sub = ctx.Subcommand
				arguments = arguments[1:]
				// start = 1
				break
			}
		}
	}

	// Can't find the command, look for subcommands if len(args) has a 2nd
	// entry.
	if cmd == nil {
		for _, s := range ctx.subcommands {
			if s.Command != parts[0] {
				continue
			}

			// Check if plumb:
			if s.plumb {
				cmd = s.Commands[0]
				sub = s
				arguments = arguments[1:]
				// start = 1
				break
			}

			// There's no second argument, so we can only look for Plumbed
			// subcommands.
			if len(parts) < 2 {
				continue
			}

			for _, c := range s.Commands {
				if c.Command == parts[1] {
					cmd = c
					sub = s
					arguments = arguments[2:]
					break
					// start = 2
				}
			}

			if cmd == nil {
				if s.QuietUnknownCommand {
					return nil
				}

				return &ErrUnknownCommand{
					Command: parts[1],
					Parent:  parts[0],
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
			Command: parts[0],
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
	var argc int

	// the last argument in the list, not used until set
	var last Argument

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.Arguments) < 1 {
		goto Call
	}

	// Argument count check.
	if argdelta := len(arguments) - len(cmd.Arguments); argdelta != 0 {
		var err error // no err if nil

		// If the function is variadic, then we can allow the last argument to
		// be empty.
		if cmd.Variadic {
			argdelta++
		}

		switch {
		// If there aren't enough arguments given.
		case argdelta < 0:
			err = ErrNotEnoughArgs

		// If there are too many arguments, then check if the command supports
		// variadic arguments. We already did a length check above.
		case argdelta > 0 && !cmd.Variadic:
			// If it's not variadic, then we can't accept it.
			err = ErrTooManyArgs
		}

		if err != nil {
			return &ErrInvalidUsage{
				Prefix: pf,
				Args:   parts,
				Index:  len(parts) - 1,
				Wrap:   err,
				Ctx:    cmd,
			}
		}
	}

	// The last argument in the arguments slice.
	last = cmd.Arguments[len(cmd.Arguments)-1]

	// Allocate a new slice the length of function arguments.
	argc = len(cmd.Arguments) - 1         // arg len without last
	argv = make([]reflect.Value, 0, argc) // could be 0

	// Parse all arguments except for the last one.
	for i := 0; i < argc; i++ {
		v, err := cmd.Arguments[i].fn(arguments[0])
		if err != nil {
			return &ErrInvalidUsage{
				Prefix: pf,
				Args:   parts,
				Index:  len(parts) - len(arguments) + i,
				Wrap:   err,
				Ctx:    cmd,
			}
		}

		// Pop arguments.
		arguments = arguments[1:]
		argv = append(argv, v)
	}

	// Is this last argument actually a variadic slice? If yes, then it
	// should still have fn normally.
	if last.fn != nil {
		// Allocate a new slice to append into.
		vars := make([]reflect.Value, 0, len(arguments))

		// Parse the rest with variadic arguments. Go's reflect states that
		// varidic parameters will automatically be copied, which is good.
		for i := 0; len(arguments) > 0; i++ {
			v, err := last.fn(arguments[0])
			if err != nil {
				return &ErrInvalidUsage{
					Prefix: pf,
					Args:   parts,
					Index:  len(parts) - len(arguments) + i,
					Wrap:   err,
					Ctx:    cmd,
				}
			}

			arguments = arguments[1:]
			vars = append(vars, v)
		}

		argv = append(argv, vars...)

	} else {
		// Create a zero value instance of this:
		v := reflect.New(last.rtype)
		var err error // return error

		switch {
		// If the argument wants all arguments:
		case last.manual != nil:
			// Call the manual parse method:
			_, err = callWith(last.manual.Func, v, reflect.ValueOf(arguments))

		// If the argument wants all arguments in string:
		case last.custom != nil:
			// Manual string seeking is a must here. This is because the string
			// could contain multiple whitespaces, and the parser would not
			// count them.
			var seekTo = cmd.Command
			// If plumbed, then there would only be the subcommand.
			if sub.plumb {
				seekTo = sub.Command
			}

			// Seek to the string.
			if i := strings.Index(content, seekTo); i > -1 {
				// Seek past the substring.
				i += len(seekTo)
				content = strings.TrimSpace(content[i:])
			}

			// Call the method with the raw unparsed command:
			_, err = callWith(last.custom.Func, v, reflect.ValueOf(content))
		}

		// Check the returned error:
		if err != nil {
			return err
		}

		// Check if the argument wants a non-pointer:
		if last.pointer {
			v = v.Elem()
		}

		// Add the argument into argv.
		argv = append(argv, v)
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

	var channelID = infer.ChannelID(ev)
	if !channelID.Valid() {
		return false
	}

	var userID = infer.UserID(ev)
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

	var channelID = infer.ChannelID(ev)
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

	var callargs = make([]reflect.Value, 0, 1+len(values))

	if v, ok := ev.(reflect.Value); ok {
		callargs = append(callargs, v)
	} else {
		callargs = append(callargs, reflect.ValueOf(ev))
	}

	callargs = append(callargs, values...)

	return errorReturns(caller.Call(callargs))
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
