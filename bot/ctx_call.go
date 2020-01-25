package bot

import (
	"encoding/csv"
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func (ctx *Context) filterEventType(evT reflect.Type) []*CommandContext {
	var callers []*CommandContext
	var middles []*CommandContext
	var found bool

	for _, cmd := range ctx.Events {
		// Inherit parent's flags
		cmd.Flag |= ctx.Flag

		// Check if middleware
		if cmd.Flag.Is(Middleware) {
			continue
		}

		if cmd.event == evT {
			callers = append(callers, cmd)
			found = true
		}
	}

	if found {
		// Search for middlewares with the same type:
		for _, mw := range ctx.mwMethods {
			if mw.event == evT {
				middles = append(middles, mw)
			}
		}
	}

	for _, sub := range ctx.subcommands {
		// Reset found status
		found = false

		for _, cmd := range sub.Events {
			// Inherit parent's flags
			cmd.Flag |= sub.Flag

			// Check if middleware
			if cmd.Flag.Is(Middleware) {
				continue
			}

			if cmd.event == evT {
				callers = append(callers, cmd)
				found = true
			}
		}

		if found {
			// Search for middlewares with the same type:
			for _, mw := range sub.mwMethods {
				if mw.event == evT {
					middles = append(middles, mw)
				}
			}
		}
	}

	return append(middles, callers...)
}

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	if evT == typeMessageCreate {
		// safe assertion always
		return ctx.callMessageCreate(ev.(*gateway.MessageCreateEvent))
	}

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
		if err := callWith(c.value, ev); err != nil {
			ctx.ErrorLogger(err)
		}
	}

	// We call the messages later, since Hidden handlers will go into the Events
	// slice, but we don't want to ignore those handlers either.
	if evT == typeMessageCreate {
		// safe assertion always
		return ctx.callMessageCreate(ev.(*gateway.MessageCreateEvent))
	}

	return nil
}

func (ctx *Context) callMessageCreate(mc *gateway.MessageCreateEvent) error {
	// check if prefix
	if !strings.HasPrefix(mc.Content, ctx.Prefix) {
		// not a command, ignore
		return nil
	}

	// trim the prefix before splitting, this way multi-words prefices work
	content := mc.Content[len(ctx.Prefix):]

	if content == "" {
		return nil // just the prefix only
	}

	// parse arguments
	args, err := ParseArgs(content)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return nil // ???
	}

	var cmd *CommandContext
	var sub *Subcommand
	var start int // arg starts from $start

	// Search for the command
	for _, c := range ctx.Commands {
		if c.Command == args[0] {
			cmd = c
			sub = ctx.Subcommand
			start = 1
			break
		}
	}

	// Can't find command, look for subcommands of len(args) has a 2nd
	// entry.
	if cmd == nil && len(args) > 1 {
		for _, s := range ctx.subcommands {
			if s.Command != args[0] {
				continue
			}

			for _, c := range s.Commands {
				if c.Command == args[1] {
					cmd = c
					sub = s
					start = 2

					// OR the flags
					c.Flag |= s.Flag
				}
			}

			if cmd == nil {
				return &ErrUnknownCommand{
					Command: args[1],
					Parent:  args[0],
					Prefix:  ctx.Prefix,
					ctx:     s.Commands,
				}
			}

			break
		}
	}

	if cmd == nil || start == 0 {
		return &ErrUnknownCommand{
			Command: args[0],
			Prefix:  ctx.Prefix,
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
	if len(cmd.Arguments) == 0 {
		goto Call
	}

	// Check manual parser
	if cmd.Arguments[0].fn == nil {
		if len(args[start:]) == 0 {
			return &ErrInvalidUsage{
				Args:   args,
				Prefix: ctx.Prefix,
				Index:  len(args) - start,
				Err:    "Not enough arguments given",
				Ctx:    cmd,
			}
		}

		// Create a zero value instance of this:
		v := reflect.New(cmd.Arguments[0].Type)

		// Pop out the subcommand name:
		args = args[1:]

		// Call the manual parse method:
		ret := cmd.Arguments[0].manual.Func.Call([]reflect.Value{
			v, reflect.ValueOf(args),
		})

		// Check the method returns for error:
		if err := errorReturns(ret); err != nil {
			// TODO: maybe wrap this?
			return err
		}

		// Add the pointer to the argument into argv:
		argv = append(argv, v)
		goto Call
	}

	// Not enough arguments given
	if len(args[start:]) != len(cmd.Arguments) {
		return &ErrInvalidUsage{
			Args:   args,
			Prefix: ctx.Prefix,
			Index:  len(args) - 1,
			Err:    "Not enough arguments given",
			Ctx:    cmd,
		}
	}

	argv = make([]reflect.Value, len(cmd.Arguments))

	for i := start; i < len(args); i++ {
		v, err := cmd.Arguments[i-start].fn(args[i])
		if err != nil {
			return &ErrInvalidUsage{
				Args:   args,
				Prefix: ctx.Prefix,
				Index:  i,
				Err:    err.Error(),
				Ctx:    cmd,
			}
		}

		argv[i-start] = v
	}

Call:
	// Try calling all middlewares first. We don't need to stack middlewares, as
	// there will only be one command match.
	for _, mw := range sub.mwMethods {
		if err := callWith(mw.value, mc); err != nil {
			return err
		}
	}

	// call the function and parse the error return value
	return callWith(cmd.value, mc, argv...)
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

func callWith(caller reflect.Value, ev interface{}, values ...reflect.Value) error {
	return errorReturns(caller.Call(append(
		[]reflect.Value{reflect.ValueOf(ev)},
		values...,
	)))
}

var ParseArgs = func(args string) ([]string, error) {
	// TODO: make modular
	// TODO: actual tokenizer+parser
	r := csv.NewReader(strings.NewReader(args))
	r.Comma = ' '

	return r.Read()
}

func errorReturns(returns []reflect.Value) error {
	// assume first is always error, since we checked for this in parseCommands
	v := returns[0].Interface()

	if v == nil {
		return nil
	}

	return v.(error)
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
