package bot

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
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
		m, err := ctx.Cabinet.Message(up.ChannelID, up.ID)
		if err != nil {
			// It's probably safe to ignore this.
			return nil
		}

		// Treat the message update as a message create event to avoid breaking
		// changes.
		msc = &gateway.MessageCreateEvent{Message: *m, Member: up.Member}

		// Fill up member, if available.
		if m.GuildID.IsValid() && up.Member == nil {
			if mem, err := ctx.Cabinet.Member(m.GuildID, m.Author.ID); err == nil {
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

func (ctx *Context) callMessageCreate(
	mc *gateway.MessageCreateEvent, value reflect.Value) error {

	v, err := ctx.callMessageCreateNoReply(mc, value)
	if err == nil && v == nil {
		return nil
	}

	if err != nil && !ctx.ReplyError && ctx.ErrorReplier == nil {
		return err
	}

	var data api.SendMessageData

	if err != nil {
		if ctx.ErrorReplier != nil {
			data = ctx.ErrorReplier(err, mc)
		} else {
			data.Content = ctx.FormatError(err)
		}
	} else {
		switch v := v.(type) {
		case string:
			data.Content = v
		case *discord.Embed:
			data.Embed = v
		case *api.SendMessageData:
			data = *v
		default:
			return nil
		}
	}

	if data.Reference == nil {
		data.Reference = &discord.MessageReference{MessageID: mc.ID}
	}

	if data.AllowedMentions == nil {
		// Do not mention on reply by default. Only allow author mentions.
		data.AllowedMentions = &api.AllowedMentions{
			Users:       []discord.UserID{mc.Author.ID},
			RepliedUser: option.False,
		}
	}

	_, err = ctx.SendMessageComplex(mc.ChannelID, data)
	return err
}

func (ctx *Context) callMessageCreateNoReply(
	mc *gateway.MessageCreateEvent, value reflect.Value) (interface{}, error) {

	// check if bot
	if !ctx.AllowBot && mc.Author.Bot {
		return nil, nil
	}

	// check if prefix
	pf, ok := ctx.HasPrefix(mc)
	if !ok {
		return nil, nil
	}

	// trim the prefix before splitting, this way multi-words prefixes work
	content := mc.Content[len(pf):]

	if content == "" {
		return nil, nil // just the prefix only
	}

	// parse arguments
	parts, parseErr := ctx.ParseArgs(content)
	// We're not checking parse errors yet, as raw arguments may be able to
	// ignore it.

	if len(parts) == 0 {
		return nil, parseErr
	}

	// Find the command and subcommand.
	commandCtx, err := ctx.findCommand(parts)
	if err != nil {
		return nil, errNoBreak(err)
	}

	var (
		arguments = commandCtx.parts
		cmd       = commandCtx.method
		sub       = commandCtx.subcmd
		plumbed   = commandCtx.plumbed
	)

	// We don't run the subcommand's middlewares here, as the callCmd function
	// already handles that.

	// Run command middlewares.
	if err := cmd.walkMiddlewares(value); err != nil {
		return nil, errNoBreak(err)
	}

	// Start converting
	var argv []reflect.Value
	var argc int

	// the last argument in the list, not used until set
	var last Argument

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.Arguments) == 0 {
		return cmd.call(value, argv...)
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
			return nil, &InvalidUsageError{
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
			return nil, &InvalidUsageError{
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
		// variadic parameters will automatically be copied, which is good.
		for i := 0; len(arguments) > 0; i++ {
			v, err := last.fn(arguments[0])
			if err != nil {
				return nil, &InvalidUsageError{
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
		var err error // return nil, error

		switch {
		// If the argument wants all arguments:
		case last.manual != nil:
			// Call the manual parse method:
			err = last.manual(v.Interface().(ManualParser), arguments)

		// If the argument wants all arguments in string:
		case last.custom != nil:
			// Ignore parser errors. This allows custom commands sliced away to
			// have erroneous hanging quotes.
			parseErr = nil

			content = trimPrefixStringAndSlice(content, sub.Command, sub.Aliases)

			// If the current command is not the plumbed command, then we can
			// keep trimming. We have to check for this, as a plumbed subcommand
			// may return nil, other non-plumbed commands.
			if !plumbed {
				content = trimPrefixStringAndSlice(content, cmd.Command, cmd.Aliases)
			}

			// Call the method with the raw unparsed command:
			err = last.custom(v.Interface().(CustomParser), content)
		}

		// Check the returned error:
		if err != nil {
			return nil, err
		}

		// Check if the argument wants a non-pointer:
		if last.pointer {
			v = v.Elem()
		}

		// Add the argument into argv.
		argv = append(argv, v)
	}

	// Check for parsing errors after parsing arguments.
	if parseErr != nil {
		return nil, parseErr
	}

	return cmd.call(value, argv...)
}

// commandContext contains related command values to call one. It is returned
// from findCommand.
type commandContext struct {
	parts   []string
	plumbed bool
	method  *MethodContext
	subcmd  *Subcommand
}

var emptyCommand = commandContext{}

// findCommand filters.
func (ctx *Context) findCommand(parts []string) (commandContext, error) {
	// Main command entrypoint cannot have plumb.
	for _, c := range ctx.Commands {
		if searchStringAndSlice(parts[0], c.Command, c.Aliases) {
			return commandContext{parts[1:], false, c, ctx.Subcommand}, nil
		}
	}

	// Can't find the command, look for subcommands if len(args) has a 2nd
	// entry.
	for _, s := range ctx.subcommands {
		if !searchStringAndSlice(parts[0], s.Command, s.Aliases) {
			continue
		}

		// The new plumbing behavior allows other commands to co-exist with a
		// plumbed command. Those commands will override the second argument,
		// similarly to a non-plumbed command.

		if len(parts) >= 2 {
			for _, c := range s.Commands {
				if searchStringAndSlice(parts[1], c.Command, c.Aliases) {
					return commandContext{parts[2:], false, c, s}, nil
				}
			}
		}

		if s.IsPlumbed() {
			return commandContext{parts[1:], true, s.plumbed, s}, nil
		}

		// If unknown command is disabled or the subcommand is hidden:
		if ctx.SilentUnknown.Subcommand || s.Hidden {
			return emptyCommand, Break
		}

		return emptyCommand, newErrUnknownCommand(s, parts)
	}

	if ctx.SilentUnknown.Command {
		return emptyCommand, Break
	}

	return emptyCommand, newErrUnknownCommand(ctx.Subcommand, parts)
}

// searchStringAndSlice searches if str is equal to isString or any of the given
// otherStrings. It is used for alias matching.
func searchStringAndSlice(str string, isString string, otherStrings []string) bool {
	if str == isString {
		return true
	}

	for _, other := range otherStrings {
		if other == str {
			return true
		}
	}

	return false
}

// trimPrefixStringAndSlice behaves similarly to searchStringAndSlice, but it
// trims the prefix and the surrounding spaces after a match.
func trimPrefixStringAndSlice(str string, prefix string, prefixes []string) string {
	if strings.HasPrefix(str, prefix) {
		return strings.TrimSpace(str[len(prefix):])
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return strings.TrimSpace(str[len(prefix):])
		}
	}

	return str
}

func errNoBreak(err error) error {
	if errors.Is(err, Break) {
		return nil
	}
	return err
}
