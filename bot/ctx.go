package bot

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/pkg/errors"
)

// TODO: add variadic arguments

type Context struct {
	*Subcommand
	*state.State

	// Descriptive (but optional) bot name
	Name string

	// Descriptive help body
	Description string

	// The prefix for commands
	Prefix string

	// FormatError formats any errors returned by anything, including the method
	// commands or the reflect functions. This also includes invalid usage
	// errors or unknown command errors. Returning an empty string means
	// ignoring the error.
	FormatError func(error) string

	// ErrorLogger logs any error that anything makes and the library can't
	// reply to the client. This includes any event callback errors that aren't
	// Message Create.
	ErrorLogger func(error)

	// ReplyError when true replies to the user the error.
	ReplyError bool

	// Subcommands contains all the registered subcommands.
	Subcommands []*Subcommand
}

// Start quickly starts a bot with the given command. It will prepend "Bot"
// into the token automatically. Refer to example/ for usage.
func Start(token string, cmd interface{},
	opts func(*Context) error) (stop func() error, err error) {

	s, err := state.New("Bot " + token)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a dgo session")
	}

	c, err := New(s, cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create rfrouter")
	}

	s.ErrorLog = func(err error) {
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
		cancel()
		return s.Close()
	}, nil
}

// Wait is a convenient function that blocks until a SIGINT is sent.
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
// Commands' exported methods will all be used as commands. Messages are parsed
// with its first argument (the command) mapped accordingly to c.MapName, which
// capitalizes the first letter automatically to reflect the exported method
// name.
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
		Prefix:     "~",
		FormatError: func(err error) string {
			return err.Error()
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
	for _, sub := range ctx.Subcommands {
		if sub.name == s.name {
			return nil, errors.New(
				"New subcommand has duplicate name: " + s.name)
		}
	}

	ctx.Subcommands = append(ctx.Subcommands, s)
	return s, nil
}

// Start adds itself into the discordgo Session handlers. This needs to be run.
// The returned function is a delete function, which removes itself from the
// Session handlers.
func (ctx *Context) Start() func() {
	return ctx.Session.AddHandler(func(v interface{}) {
		if err := ctx.callCmd(v); err != nil {
			if str := ctx.FormatError(err); str != "" {
				// Log the main error first
				ctx.ErrorLogger(errors.Wrap(err, str))

				mc, ok := v.(*gateway.MessageCreateEvent)
				if !ok {
					return
				}

				if ctx.ReplyError {
					_, Merr := ctx.SendMessage(mc.ChannelID, str, nil)
					if Merr != nil {
						// Then the message error
						ctx.ErrorLogger(Merr)
						// TODO: there ought to be a better way lol
					}
				}
			}
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
	var help strings.Builder

	// Generate the headers and descriptions
	help.WriteString("__Help__")

	if ctx.Name != "" {
		help.WriteString(": " + ctx.Name)
	}

	if ctx.Description != "" {
		help.WriteString("\n      " + ctx.Description)
	}

	if ctx.Flag.Is(AdminOnly) {
		// That's it.
		return help.String()
	}

	// Separators
	help.WriteString("\n---\n")

	// Generate all commands
	help.WriteString("__Commands__\n")

	for _, cmd := range ctx.Commands {
		if cmd.Flag.Is(AdminOnly) {
			// Hidden
			continue
		}

		help.WriteString("      " + ctx.Prefix + cmd.Name())

		switch {
		case len(cmd.Usage()) > 0:
			help.WriteString(" " + strings.Join(cmd.Usage(), " "))
		case cmd.Description != "":
			help.WriteString(": " + cmd.Description)
		}

		help.WriteByte('\n')
	}

	var subHelp = strings.Builder{}

	for _, sub := range ctx.Subcommands {
		if sub.Flag.Is(AdminOnly) {
			// Hidden
			continue
		}

		subHelp.WriteString("      " + sub.Name())

		if sub.Description != "" {
			subHelp.WriteString(": " + sub.Description)
		}

		subHelp.WriteByte('\n')

		for _, cmd := range sub.Commands {
			if cmd.Flag.Is(AdminOnly) {
				continue
			}

			subHelp.WriteString("            " +
				ctx.Prefix + sub.Name() + " " + cmd.Name())

			switch {
			case len(cmd.Usage()) > 0:
				subHelp.WriteString(" " + strings.Join(cmd.Usage(), " "))
			case cmd.Description != "":
				subHelp.WriteString(": " + cmd.Description)
			}

			subHelp.WriteByte('\n')
		}
	}

	if sub := subHelp.String(); sub != "" {
		help.WriteString("---\n")
		help.WriteString("__Subcommands__\n")
		help.WriteString(sub)
	}

	return help.String()
}
