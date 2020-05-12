package bot

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/bot/extras/shellwords"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/pkg/errors"
)

// Prefixer checks a message if it starts with the desired prefix. By default,
// NewPrefix() is used.
type Prefixer func(*gateway.MessageCreateEvent) (prefix string, ok bool)

// NewPrefix creates a simple prefix checker using strings. As the default
// prefix is "!", the function is called as NewPrefix("!").
func NewPrefix(prefixes ...string) Prefixer {
	return func(msg *gateway.MessageCreateEvent) (string, bool) {
		for _, prefix := range prefixes {
			if strings.HasPrefix(msg.Content, prefix) {
				return prefix, true
			}
		}
		return "", false
	}
}

// ArgsParser is the function type for parsing message content into fields,
// usually delimited by spaces.
type ArgsParser func(content string) ([]string, error)

// DefaultArgsParser implements a parser similar to that of shell's,
// implementing quotes as well as escapes.
func DefaultArgsParser() ArgsParser {
	return shellwords.Parse
}

// Context is the bot state for commands and subcommands.
//
// Commands
//
// A command can be created by making it a method of Commands, or whatever
// struct was given to the constructor. This following example creates a command
// with a single integer argument (which can be ran with "~example 123"):
//
//    func (c *Commands) Example(
//        m *gateway.MessageCreateEvent, i int) (string, error) {
//
//        return fmt.Sprintf("You sent: %d", i)
//    }
//
// Commands' exported methods will all be used as commands. Messages are parsed
// with its first argument (the command) mapped accordingly to c.MapName, which
// capitalizes the first letter automatically to reflect the exported method
// name.
//
// A command can either return either an error, or data and error. The only data
// types allowed are string, *discord.Embed, and *api.SendMessageData. Any other
// return types will invalidate the method.
//
// Events
//
// An event can only have one argument, which is the pointer to the event
// struct. It can also only return error.
//
//    func (c *Commands) Example(o *gateway.TypingStartEvent) error {
//        log.Println("Someone's typing!")
//        return nil
//    }
type Context struct {
	*Subcommand
	*state.State

	// Descriptive (but optional) bot name
	Name string

	// Descriptive help body
	Description string

	// Called to parse message content, default to DefaultArgsParser().
	ParseArgs ArgsParser

	// Called to check a message's prefix. The default prefix is "!". Refer to
	// NewPrefix().
	HasPrefix Prefixer

	// AllowBot makes the router also process MessageCreate events from bots.
	// This is false by default and only applies to MessageCreate.
	AllowBot bool

	// FormatError formats any errors returned by anything, including the method
	// commands or the reflect functions. This also includes invalid usage
	// errors or unknown command errors. Returning an empty string means
	// ignoring the error.
	//
	// By default, this field replaces all @ with @\u200b, which prevents an
	// @everyone mention.
	FormatError func(error) string

	// ErrorLogger logs any error that anything makes and the library can't
	// reply to the client. This includes any event callback errors that aren't
	// Message Create.
	ErrorLogger func(error)

	// ReplyError when true replies to the user the error. This only applies to
	// MessageCreate events.
	ReplyError bool

	// Subcommands contains all the registered subcommands. This is not
	// exported, as it shouldn't be used directly.
	subcommands []*Subcommand

	// Quick access map from event types to pointers. This map will never have
	// MessageCreateEvent's type.
	typeCache sync.Map // map[reflect.Type][]*CommandContext
}

// Start quickly starts a bot with the given command. It will prepend "Bot"
// into the token automatically. Refer to example/ for usage.
func Start(token string, cmd interface{},
	opts func(*Context) error) (wait func() error, err error) {

	s, err := state.New("Bot " + token)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a dgo session")
	}

	c, err := New(s, cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create rfrouter")
	}

	s.Gateway.ErrorLog = func(err error) {
		c.ErrorLogger(err)
	}

	if opts != nil {
		if err := opts(c); err != nil {
			return nil, err
		}
	}

	cancel := c.Start()

	if err := s.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return func() error {
		Wait()
		// remove handler first
		cancel()
		// then finish closing session
		return s.Close()
	}, nil
}

// Wait blocks until SIGINT.
func Wait() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)
	<-sigs
}

// New makes a new context with a "~" as the prefix. cmds must be a pointer to a
// struct with a *Context field. Example:
//
//    type Commands struct {
//        Ctx *Context
//    }
//
//    cmds := &Commands{}
//    c, err := rfrouter.New(session, cmds)
//
// The default prefix is "~", which means commands must start with "~" followed
// by the command name in the first argument, else it will be ignored.
//
// c.Start() should be called afterwards to actually handle incoming events.
func New(s *state.State, cmd interface{}) (*Context, error) {
	c, err := NewSubcommand(cmd)
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Subcommand: c,
		State:      s,
		ParseArgs:  DefaultArgsParser(),
		HasPrefix:  NewPrefix("~"),
		FormatError: func(err error) string {
			// Escape all pings, including @everyone.
			return strings.Replace(err.Error(), "@", "@\u200b", -1)
		},
		ErrorLogger: func(err error) {
			log.Println("Bot error:", err)
		},
		ReplyError: true,
	}

	if err := ctx.InitCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize with given cmds")
	}

	return ctx, nil
}

// Subcommands returns the slice of subcommands. To add subcommands, use
// RegisterSubcommand().
func (ctx *Context) Subcommands() []*Subcommand {
	// Getter is not useless, refer to the struct doc for reason.
	return ctx.subcommands
}

// FindCommand finds a command based on the struct and method name. The queried
// names will have their flags stripped.
//
// Example
//
//    // Find a command from the main context:
//    cmd := ctx.FindCommand("", "Method")
//    // Find a command from a subcommand:
//    cmd  = ctx.FindCommand("Starboard", "Reset")
//
func (ctx *Context) FindCommand(structname, methodname string) *CommandContext {
	if structname == "" {
		for _, c := range ctx.Commands {
			if c.MethodName == methodname {
				return c
			}
		}

		return nil
	}

	for _, sub := range ctx.subcommands {
		if sub.StructName != structname {
			continue
		}

		for _, c := range sub.Commands {
			if c.MethodName == methodname {
				return c
			}
		}
	}

	return nil
}

// MustRegisterSubcommand tries to register a subcommand, and will panic if it
// fails. This is recommended, as subcommands won't change after initializing
// once in runtime, thus fairly harmless after development.
func (ctx *Context) MustRegisterSubcommand(cmd interface{}) *Subcommand {
	s, err := ctx.RegisterSubcommand(cmd)
	if err != nil {
		panic(err)
	}

	return s
}

// RegisterSubcommand registers and adds cmd to the list of subcommands. It will
// also return the resulting Subcommand.
func (ctx *Context) RegisterSubcommand(cmd interface{}) (*Subcommand, error) {
	s, err := NewSubcommand(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to add subcommand")
	}

	// Register the subcommand's name.
	s.NeedsName()

	if err := s.InitCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize subcommand")
	}

	// Do a collision check
	for _, sub := range ctx.subcommands {
		if sub.Command == s.Command {
			return nil, errors.New(
				"New subcommand has duplicate name: " + s.Command)
		}
	}

	ctx.subcommands = append(ctx.subcommands, s)
	return s, nil
}

// Start adds itself into the discordgo Session handlers. This needs to be run.
// The returned function is a delete function, which removes itself from the
// Session handlers.
func (ctx *Context) Start() func() {
	return ctx.State.AddHandler(func(v interface{}) {
		err := ctx.callCmd(v)
		if err == nil {
			return
		}

		str := ctx.FormatError(err)
		if str == "" {
			return
		}

		mc, isMessage := v.(*gateway.MessageCreateEvent)

		// Log the main error if reply is disabled or if the event isn't a
		// message.
		if !ctx.ReplyError || !isMessage {
			// Ignore trivial errors:
			switch err.(type) {
			case *ErrInvalidUsage, *ErrUnknownCommand:
				// Ignore
			default:
				ctx.ErrorLogger(errors.Wrap(err, "Command error"))
			}

			return
		}

		// Only reply if the event is not a message.
		if !isMessage {
			return
		}

		_, err = ctx.SendMessageComplex(mc.ChannelID, api.SendMessageData{
			// Escape the error using the message sanitizer:
			Content: ctx.SanitizeMessage(str),
			AllowedMentions: &api.AllowedMentions{
				// Don't allow mentions.
				Parse: []api.AllowedMentionType{},
			},
		})
		if err != nil {
			ctx.ErrorLogger(err)

			// TODO: there ought to be a better way lol
		}
	})
}

// Call should only be used if you know what you're doing.
func (ctx *Context) Call(event interface{}) error {
	return ctx.callCmd(event)
}

// Help generates one. This function is used more for reference than an actual
// help message. As such, it only uses exported fields or methods.
func (ctx *Context) Help() string {
	return ctx.help(true)
}

func (ctx *Context) HelpAdmin() string {
	return ctx.help(false)
}

func (ctx *Context) help(hideAdmin bool) string {
	const indent = "      "

	var help strings.Builder

	// Generate the headers and descriptions
	help.WriteString("__Help__")

	if ctx.Name != "" {
		help.WriteString(": " + ctx.Name)
	}

	if ctx.Description != "" {
		help.WriteString("\n" + indent + ctx.Description)
	}

	if ctx.Flag.Is(AdminOnly) {
		// That's it.
		return help.String()
	}

	// Separators
	help.WriteString("\n---\n")

	// Generate all commands
	help.WriteString("__Commands__")
	help.WriteString(ctx.Subcommand.Help(indent, hideAdmin))
	help.WriteByte('\n')

	var subHelp = strings.Builder{}
	var subcommands = ctx.Subcommands()

	for _, sub := range subcommands {
		if help := sub.Help(indent, hideAdmin); help != "" {
			for _, line := range strings.Split(help, "\n") {
				subHelp.WriteString(indent)
				subHelp.WriteString(line)
				subHelp.WriteByte('\n')
			}
		}
	}

	if subHelp.Len() > 0 {
		help.WriteString("---\n")
		help.WriteString("__Subcommands__\n")
		help.WriteString(subHelp.String())
	}

	return help.String()
}
