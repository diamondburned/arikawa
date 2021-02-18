package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/bot/extras/shellwords"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state"
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

	// QuietUnknownCommand, if true, will not make the bot reply with an unknown
	// command error into the chat. This will apply to all other subcommands.
	// SilentUnknown controls whether or not an ErrUnknownCommand should be
	// returned (instead of a silent error).
	SilentUnknown struct {
		// Command when true will silent only unknown commands. Known
		// subcommands with unknown commands will still error out.
		Command bool
		// Subcommand when true will suppress unknown subcommands.
		Subcommand bool
	}

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

	// ErrorReplier is an optional function that allows changing how the error
	// is replied. It overrides ReplyError and is only used for MessageCreate
	// events.
	//
	// Note that errors that are passed in here will bypas FormatError; in other
	// words, the implementation might only care about ErrorReplier and leave
	// FormatError as it is.
	ErrorReplier func(err error, src *gateway.MessageCreateEvent) api.SendMessageData

	// EditableCommands when true will also listen for MessageUpdateEvent and
	// treat them as newly created messages. This is convenient if you want
	// to quickly edit a message and re-execute the command.
	EditableCommands bool

	// Subcommands contains all the registered subcommands. This is not
	// exported, as it shouldn't be used directly.
	subcommands []*Subcommand

	// Quick access map from event types to pointers. This map will never have
	// MessageCreateEvent's type.
	typeCache sync.Map // map[reflect.Type][]*CommandContext

	// commands contains all known commands. It maps CommandIDs.
	commands sync.Map
}

// Start quickly starts a bot with the given command. It will prepend "Bot"
// into the token automatically. Refer to example/ for usage.
func Start(
	token string, cmd interface{},
	opts func(*Context) error) (wait func() error, err error) {

	if token == "" {
		return nil, errors.New("token is not given")
	}

	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	s, err := state.New(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a dgo session")
	}

	// fail api request if they (will) take up more than 5 minutes
	s.Client.Client.Timeout = 5 * time.Minute

	c, err := New(s, cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rfrouter")
	}

	s.Gateway.ErrorLog = func(err error) {
		c.ErrorLogger(err)
	}

	if opts != nil {
		if err := opts(c); err != nil {
			return nil, err
		}
	}

	c.AddIntents(c.DeriveIntents())
	c.AddIntents(gateway.IntentGuilds) // for channel event caching

	cancel := c.Start()

	if err := s.Open(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to Discord")
	}

	return func() error {
		Wait()
		// remove handler first
		cancel()
		// then finish closing session
		return s.Close()
	}, nil
}

// Run starts the bot, prints a message into the console, and blocks until
// SIGINT. "Bot" is prepended into the token automatically, similar to Start.
// The function will call os.Exit(1) on an initialization or cleanup error.
func Run(token string, cmd interface{}, opts func(*Context) error) {
	wait, err := Start(token, cmd, opts)
	if err != nil {
		log.Fatalln("failed to start:", err)
	}

	log.Println("Bot is running.")

	if err := wait(); err != nil {
		log.Fatalln("cleanup error:", err)
	}
}

// Wait blocks until SIGINT.
func Wait() {
	sigs := make(chan os.Signal, 1)
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
//    c, err := bot.New(session, cmds)
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
		return nil, errors.Wrap(err, "failed to initialize with given cmds")
	}

	return ctx, nil
}

// AddIntents adds the given Gateway Intent into the Gateway. This is a
// convenient function that calls Gateway's AddIntent.
func (ctx *Context) AddIntents(i gateway.Intents) {
	ctx.Gateway.AddIntents(i)
}

// DeriveIntents derives all possible gateway intents from this context and all
// its subcommands' method handlers and middlewares.
func (ctx *Context) DeriveIntents() gateway.Intents {
	var intents = ctx.Subcommand.DeriveIntents()
	for _, subcmd := range ctx.subcommands {
		intents |= subcmd.DeriveIntents()
	}
	return intents
}

// Subcommands returns the slice of subcommands. To add subcommands, use
// RegisterSubcommand().
func (ctx *Context) Subcommands() []*Subcommand {
	// Getter is not useless, refer to the struct doc for reason.
	return ctx.subcommands
}

// FindMethod finds a method based on the struct and method name. The queried
// names will have their flags stripped.
//
//    // Find a command from the main context:
//    cmd := ctx.FindMethod("", "Method")
//    // Find a command from a subcommand:
//    cmd  = ctx.FindMethod("Starboard", "Reset")
//
func (ctx *Context) FindCommand(structName, methodName string) *MethodContext {
	if structName == "" {
		return ctx.Subcommand.FindCommand(methodName)
	}
	for _, sub := range ctx.subcommands {
		if sub.StructName == structName {
			return sub.FindCommand(methodName)
		}
	}
	return nil
}

// MustRegisterSubcommand tries to register a subcommand, and will panic if it
// fails. This is recommended, as subcommands won't change after initializing
// once in runtime, thus fairly harmless after development.
//
// If no names are given or if the first name is empty, then the subcommand name
// will be derived from the struct name. If one name is given, then that name
// will override the struct name. Any other name values will be aliases.
//
// It is recommended to use this method to add subcommand aliases over manually
// altering the Aliases slice of each Subcommand, as it does collision checks
// against other subcommands as well.
func (ctx *Context) MustRegisterSubcommand(cmd interface{}, names ...string) *Subcommand {
	s, err := ctx.RegisterSubcommand(cmd, names...)
	if err != nil {
		panic(err)
	}
	return s
}

// RegisterSubcommand registers and adds cmd to the list of subcommands. It will
// also return the resulting Subcommand. Refer to MustRegisterSubcommand for the
// names argument.
func (ctx *Context) RegisterSubcommand(cmd interface{}, names ...string) (*Subcommand, error) {
	s, err := NewSubcommand(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add subcommand")
	}

	// Register the subcommand's name.
	s.NeedsName()

	if len(names) > 0 && names[0] != "" {
		s.Command = names[0]
	}

	if len(names) > 1 {
		// Copy the slice for expected behaviors.
		s.Aliases = append([]string(nil), names[1:]...)
	}

	if err := s.InitCommands(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to initialize subcommand")
	}

	// Check if the existing command name already exists. This could really be
	// optimized, but since it's in a cold path, who cares.
	var subcommandNames = append([]string{s.Command}, s.Aliases...)

	for _, name := range subcommandNames {
		for _, sub := range ctx.subcommands {
			// Check each alias against the subcommand name.
			if sub.Command == name {
				return nil, fmt.Errorf("new subcommand has duplicate name: %q", name)
			}

			// Also check each alias against other subcommands' aliases.
			for _, subalias := range sub.Aliases {
				if subalias == name {
					return nil, fmt.Errorf("new subcommand has duplicate alias: %q", name)
				}
			}
		}
	}

	ctx.subcommands = append(ctx.subcommands, s)
	return s, nil
}

// emptyMentionTypes is used by Start() to not parse any mentions.
var emptyMentionTypes = []api.AllowedMentionType{}

// Start adds itself into the session handlers. This needs to be run. The
// returned function is a delete function, which removes itself from the
// Session handlers.
func (ctx *Context) Start() func() {
	return ctx.State.AddHandler(func(v interface{}) {
		err := ctx.Call(v)
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
				ctx.ErrorLogger(errors.Wrap(err, "command error"))
			}

			return
		}

		// Only reply if the event is not a message.
		if !isMessage {
			return
		}

		_, err = ctx.SendMessageComplex(mc.ChannelID, api.SendMessageData{
			Content: str,
			// Don't allow mentions.
			AllowedMentions: &api.AllowedMentions{Parse: emptyMentionTypes},
		})

		if err != nil {
			ctx.ErrorLogger(errors.Wrap(err, "failed to send message"))

			// TODO: there ought to be a better way lol
		}
	})
}

// Close closes the gateway gracefully. Bots that need to preserve the session
// ID after closing should NOT use this method.
func (ctx *Context) Close() error {
	return ctx.Session.CloseGracefully()
}

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

// Call calls all handlers that are subscribed to the given event. This method
// should only be used if you know what you're doing.
func (ctx *Context) Call(ev interface{}) (bottomError error) {
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

	// We call the messages later, since we want MessageCreate middlewares to
	// run as well.
	switch evT {
	case typeMessageUpdate:
		if !ctx.EditableCommands {
			return nil
		}

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
		msc := &gateway.MessageCreateEvent{Message: *m, Member: up.Member}

		// Fill up member, if available.
		if m.GuildID.IsValid() && up.Member == nil {
			if mem, err := ctx.Cabinet.Member(m.GuildID, m.Author.ID); err == nil {
				msc.Member = mem
			}
		}

		// Update the reflect value as well.
		evV = reflect.ValueOf(msc)

		// Handle this the same as a MessageCreate.
		return ctx.callMessageCreate(msc, evV)

	case typeMessageCreate:
		// There's no need for an errNoBreak here, as the method already checked
		// for that.
		return ctx.callMessageCreate(ev.(*gateway.MessageCreateEvent), evV)

	case typeInteractionCreate:
		ev := ev.(*gateway.InteractionCreateEvent)
		// Only handle interaction COMMANDs.
		if ev.Type == gateway.CommandInteraction {
			return ctx.callInteractionCreate(ev, evV)
		}
	}

	// Unknown event; ignore.
	return nil
}

func errNoBreak(err error) error {
	if errors.Is(err, Break) {
		return nil
	}
	return err
}
