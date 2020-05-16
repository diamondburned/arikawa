package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

// Break is a non-fatal error that could be returned from middlewares to stop
// the chain of execution.
var Break = errors.New("break middleware chain, non-fatal")

// filterEventType filters all commands and subcommands into a 2D slice,
// structured so that a Break would only exit out the nested slice.
func (ctx *Context) filterEventType(evT reflect.Type) (callers [][]caller) {
	// Find the main context first.
	callers = append(callers, ctx.eventCallers(evT))

	for _, sub := range ctx.subcommands {
		// Find subcommands second.
		callers = append(callers, sub.eventCallers(evT))
	}

	return
}

func (ctx *Context) callCmd(ev interface{}) (bottomError error) {
	evV := reflect.ValueOf(ev)
	evT := evV.Type()

	var callers [][]caller

	// Hit the cache
	t, ok := ctx.typeCache.Load(evT)
	if ok {
		callers = t.([][]caller)
	} else {
		callers = ctx.filterEventType(evT)
		ctx.typeCache.Store(evT, callers)
	}

	for _, subcallers := range callers {
		for _, c := range subcallers {
			_, err := c.call(evV)
			if err != nil {
				// Only count as an error if it's not Break.
				if err = errNoBreak(err); err != nil {
					bottomError = err
				}

				// Break the caller loop only for this subcommand.
				break
			}
		}
	}

	var msc *gateway.MessageCreateEvent

	// We call the messages later, since we want MessageCreate middlewares to
	// run as well.
	switch {
	case evT == typeMessageCreate:
		msc = ev.(*gateway.MessageCreateEvent)

	case evT == typeMessageUpdate && ctx.EditableCommands:
		up := ev.(*gateway.MessageUpdateEvent)
		// Message updates could have empty contents when only their embeds are
		// filled. We don't need that here.
		if up.Content == "" {
			return nil
		}

		// Query the updated message.
		m, err := ctx.Store.Message(up.ChannelID, up.ID)
		if err != nil {
			// It's probably safe to ignore this.
			return nil
		}

		// Treat the message update as a message create event to avoid breaking
		// changes.
		msc = &gateway.MessageCreateEvent{Message: *m, Member: up.Member}

		// Fill up member, if available.
		if m.GuildID.Valid() && up.Member == nil {
			if mem, err := ctx.Store.Member(m.GuildID, m.Author.ID); err == nil {
				msc.Member = mem
			}
		}

		// Update the reflect value as well.
		evV = reflect.ValueOf(msc)

	default:
		// Unknown event, return.
		return nil
	}

	// There's no need for an errNoBreak here, as the method already checked
	// for that.
	return ctx.callMessageCreate(msc, evV)
}

func (ctx *Context) callMessageCreate(mc *gateway.MessageCreateEvent, value reflect.Value) error {
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

	// Find the command and subcommand.
	arguments, cmd, sub, err := ctx.findCommand(parts)
	if err != nil {
		return errNoBreak(err)
	}

	// We don't run the subcommand's middlewares here, as the callCmd function
	// already handles that.

	// Run command middlewares.
	if err := cmd.walkMiddlewares(value); err != nil {
		return errNoBreak(err)
	}

	// Start converting
	var argv []reflect.Value
	var argc int

	// the last argument in the list, not used until set
	var last Argument

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.Arguments) == 0 {
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
			// We can't rely on the plumbing behavior.
			if sub.plumbed != nil {
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
	// call the function and parse the error return value
	v, err := cmd.call(value, argv...)
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

// findCommand filters.
func (ctx *Context) findCommand(parts []string) ([]string, *MethodContext, *Subcommand, error) {
	// Main command entrypoint cannot have plumb.
	for _, c := range ctx.Commands {
		if c.Command == parts[0] {
			return parts[1:], c, ctx.Subcommand, nil
		}
	}

	// Can't find the command, look for subcommands if len(args) has a 2nd
	// entry.
	for _, s := range ctx.subcommands {
		if s.Command != parts[0] {
			continue
		}

		// Only actually plumb if we actually have a plumbed handler AND
		//    1. We only have one command handler OR
		//    2. We only have the subcommand name but no command.
		if s.plumbed != nil && (len(s.Commands) == 1 || len(parts) <= 2) {
			return parts[1:], s.plumbed, s, nil
		}

		if len(parts) >= 2 {
			for _, c := range s.Commands {
				if c.Command == parts[1] {
					return parts[2:], c, s, nil
				}
			}
		}

		// If unknown command is disabled or the subcommand is hidden:
		if ctx.SilentUnknown.Subcommand || s.Hidden {
			return nil, nil, nil, Break
		}

		return nil, nil, nil, &ErrUnknownCommand{
			Parts:  parts,
			Subcmd: s,
		}
	}

	if ctx.SilentUnknown.Command {
		return nil, nil, nil, Break
	}

	return nil, nil, nil, &ErrUnknownCommand{
		Parts:  parts,
		Subcmd: ctx.Subcommand,
	}
}

func errNoBreak(err error) error {
	if errors.Is(err, Break) {
		return nil
	}
	return err
}
