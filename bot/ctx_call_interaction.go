package bot

import (
	"reflect"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
	"github.com/pkg/errors"
)

// interactionCommand contains the subcommand and method of each command.
type interactionCommand struct {
	*discord.Command

	subcmd *Subcommand
	method *MethodContext
}

func stringifyOptions(opts []gateway.InteractionOption) []string {
	strings := make([]string, len(opts))
	for i, opt := range opts {
		strings[i] = opt.Name + "=" + opt.Value
	}
	return strings
}

func findOption(opts []gateway.InteractionOption, n string) gateway.InteractionOption {
	for _, opt := range opts {
		if opt.Name == n {
			return opt
		}
	}
	return gateway.InteractionOption{}
}

// UpdateCommands updates global interaction commands to synchronize with the
// currently known ones from this Context.
func (ctx *Context) UpdateCommands() error {
	me, err := ctx.Me()
	if err != nil {
		return errors.Wrap(err, "failed to get current user")
	}

	appID := discord.AppID(me.ID)

	commands, err := ctx.Client.Commands(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get existing commands")
	}

	// Use a map for fast lookup.
	commandMap := make(map[string]*discord.Command, len(commands))
	for i, cmd := range commands {
		commandMap[cmd.Name] = &commands[i]
	}

	for _, method := range ctx.Commands {
		cmd, ok := commandMap[method.Command]
		if !ok && !method.equalCommand(*cmd) {
			cmd, err = ctx.CreateCommand(appID, method.constructCommand())
			if err != nil {
				return errors.Wrapf(err, "failed to recreate method %q", method.Command)
			}
		}

		ctx.commands.Store(cmd.ID, interactionCommand{
			Command: cmd,
			method:  method,
		})
	}

	for _, sub := range ctx.subcommands {
		cmd, ok := commandMap[sub.Command]
		if !ok || !sub.equalCommand(*cmd) {
			cmd, err = ctx.CreateCommand(appID, sub.constructCommand())
			if err != nil {
				return errors.Wrapf(err, "failed to recreate subcommand %q", sub.Command)
			}
		}

		ctx.commands.Store(cmd.ID, interactionCommand{
			Command: cmd,
			subcmd:  sub,
		})
	}

	return nil
}

func (ctx *Context) callInteractionCreate(
	interaction *gateway.InteractionCreateEvent, value reflect.Value) error {

	// This is possibly a redundant check; can bots even make interactions?
	if !ctx.AllowBot && interaction.Member.User.Bot {
		return nil
	}

	// Find the command and subcommand.
	cv, ok := ctx.commands.Load(interaction.Data.ID)
	if !ok {
		return newErrUnknownCommand(ctx.Subcommand, []string{interaction.Data.Name})
	}

	commandInfo := cv.(interactionCommand)

	var (
		method  *MethodContext // actual command
		subcmd  *Subcommand    // command containing subcommands
		options []gateway.InteractionOption
	)

	switch {
	case commandInfo.method != nil:
		method = commandInfo.method
		subcmd = ctx.Subcommand
		options = interaction.Data.Options

	case commandInfo.subcmd != nil:
		subcmd = commandInfo.subcmd
		if len(interaction.Data.Options) == 0 {
			return newErrUnknownCommand(subcmd, []string{interaction.Data.Name})
		}

		// This is only the subcommand; we still need to search for the method.
		for _, opt := range interaction.Data.Options {
			if opt.Value != "" || len(opt.Options) == 0 {
				continue
			}

			method = subcmd.findCommandName(opt.Name)
			if method == nil {
				continue
			}

			options = opt.Options
			break
		}

		if method == nil {
			return newErrUnknownCommand(subcmd, []string{
				interaction.Data.Name,
				interaction.Data.Options[0].Name,
			})
		}
	}

	// We don't run the subcommand's middlewares here, as the callCmd function
	// already handles that.

	// Run command middlewares.
	if err := method.walkMiddlewares(value); err != nil {
		return errNoBreak(err)
	}

	// Start converting
	var argv []reflect.Value
	var argc int

	// the last argument in the list, not used until set
	var last Argument

	// contains arguments converted to its string form
	var values []string

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(method.Arguments) == 0 {
		goto Call
	}

	// Argument count check.
	if argdelta := len(options) - len(method.Arguments); argdelta != 0 {
		var err error // no err if nil

		// If the function is variadic, then we can allow the last argument to
		// be empty.
		if method.Variadic {
			argdelta++
		}

		switch {
		// If there aren't enough arguments given.
		case argdelta < 0:
			err = ErrNotEnoughArgs

		// If there are too many arguments, then check if the command supports
		// variadic arguments. We already did a length check above.
		case argdelta > 0 && !method.Variadic:
			// If it's not variadic, then we can't accept it.
			err = ErrTooManyArgs
		}

		if err != nil {
			return &ErrInvalidUsage{
				Args:  stringifyOptions(options),
				Index: len(options) - 1,
				Wrap:  err,
				Ctx:   method,
			}
		}
	}

	// Re-sort the options and get their values.
	values = make([]string, len(options))
	for i, arg := range method.Arguments {
		opt := findOption(options, arg.String)
		if opt.Value == "" {
			return &ErrInvalidUsage{
				Args:  stringifyOptions(options),
				Index: len(options) - 1,
				Wrap:  errors.New("missing option " + arg.String),
			}
		}

		values[i] = opt.Value
	}

	// The last argument in the arguments slice.
	last = method.Arguments[len(method.Arguments)-1]

	// Allocate a new slice the length of function arguments.
	argc = len(method.Arguments) - 1      // arg len without last
	argv = make([]reflect.Value, 0, argc) // could be 0

	// Parse all arguments except for the last one.
	for i := 0; i < argc; i++ {
		v, err := method.Arguments[i].fn(options[0].Value)
		if err != nil {
			return &ErrInvalidUsage{
				Args:  stringifyOptions(options),
				Index: i,
				Wrap:  err,
				Ctx:   method,
			}
		}

		// Pop arguments.
		options = options[1:]
		argv = append(argv, v)
	}

	// Is this last argument actually a variadic slice? If yes, then it
	// should still have fn normally.
	if last.fn != nil {
		// Allocate a new slice to append into.
		vars := make([]reflect.Value, 0, len(options))

		// Parse the rest with variadic arguments. Go's reflect states that
		// variadic parameters will automatically be copied, which is good.
		for i := 0; len(options) > 0; i++ {
			v, err := last.fn(options[0].Value)
			if err != nil {
				return &ErrInvalidUsage{
					Args:  stringifyOptions(options),
					Index: i,
					Wrap:  err,
					Ctx:   method,
				}
			}

			options = options[1:]
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

			for _, opt := range options {
				arg := findArgument(method.Arguments, opt.Name)
			}
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
			// may return other non-plumbed commands.
			if !plumbed {
				content = trimPrefixStringAndSlice(content, cmd.Command, cmd.Aliases)
			}

			// Call the method with the raw unparsed command:
			err = last.custom(v.Interface().(CustomParser), content)
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

	// Check for parsing errors after parsing arguments.
	if parseErr != nil {
		return parseErr
	}

Call:
	// call the function and parse the error return value
	v, err := cmd.call(value, argv...)
	if err != nil || v == nil {
		return err
	}

	var data api.SendMessageData

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
