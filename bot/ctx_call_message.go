package bot

import (
	"reflect"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
)

// commandContext contains related command values to call one. It is returned
// from findCommand.
type commandContext struct {
	parts   []string
	plumbed bool
	method  *MethodContext
	subcmd  *Subcommand
}

var emptyCommand = commandContext{}

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
	commandCtx, err := ctx.findCommandContext(parts)
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
			return nil, &ErrInvalidUsage{
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
			return nil, &ErrInvalidUsage{
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
				return nil, &ErrInvalidUsage{
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
