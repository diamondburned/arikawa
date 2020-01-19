package bot

import (
	"encoding/csv"
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func (ctx *Context) callCmd(ev interface{}) error {
	evT := reflect.TypeOf(ev)

	if evT != typeMessageCreate {
		var callers []reflect.Value
		var isAdmin *bool // i want to die

		for _, cmd := range ctx.Commands {
			if cmd.event == evT {
				if cmd.Flag.Is(AdminOnly) &&
					!ctx.eventIsAdmin(ev, &isAdmin) {

					continue
				}

				callers = append(callers, cmd.value)
			}
		}

		for _, sub := range ctx.Subcommands {
			if sub.Flag.Is(AdminOnly) &&
				!ctx.eventIsAdmin(ev, &isAdmin) {

				continue
			}

			for _, cmd := range sub.Commands {
				if cmd.event == evT {
					if cmd.Flag.Is(AdminOnly) &&
						!ctx.eventIsAdmin(ev, &isAdmin) {

						continue
					}

					callers = append(callers, cmd.value)
				}
			}
		}

		for _, c := range callers {
			if err := callWith(c, ev); err != nil {
				ctx.ErrorLogger(err)
			}
		}

		return nil
	}

	// safe assertion always
	mc := ev.(*gateway.MessageCreateEvent)

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

	if len(args) < 1 {
		return nil // ???
	}

	var cmd *CommandContext
	var start int // arg starts from $start

	// Search for the command
	for _, c := range ctx.Commands {
		if c.name == args[0] {
			cmd = c
			start = 1
			break
		}
	}

	// Can't find command, look for subcommands of len(args) has a 2nd
	// entry.
	if cmd == nil && len(args) > 1 {
		for _, s := range ctx.Subcommands {
			if s.name != args[0] {
				continue
			}

			for _, c := range s.Commands {
				if c.name == args[1] {
					cmd = c
					start = 2
					break
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
		}
	}

	if cmd == nil || start == 0 {
		return &ErrUnknownCommand{
			Command: args[0],
			Prefix:  ctx.Prefix,
			ctx:     ctx.Commands,
		}
	}

	// Start converting
	var argv []reflect.Value

	// Check manual parser
	if cmd.parseType != nil {
		// Create a zero value instance of this
		v := reflect.New(cmd.parseType)

		// Call the manual parse method
		ret := cmd.parseMethod.Func.Call([]reflect.Value{
			v, reflect.ValueOf(args),
		})

		// Check the method returns for error
		if err := errorReturns(ret); err != nil {
			// TODO: maybe wrap this?
			return err
		}

		// Add the pointer to the argument into argv
		argv = append(argv, v)
		goto Call
	}

	// Here's an edge case: when the handler takes no arguments, we allow that
	// anyway, as they might've used the raw content.
	if len(cmd.arguments) == 0 {
		goto Call
	}

	// Not enough arguments given
	if len(args[start:]) != len(cmd.arguments) {
		return &ErrInvalidUsage{
			Args:   args,
			Prefix: ctx.Prefix,
			Index:  len(cmd.arguments) - start,
			Err:    "Not enough arguments given",
			ctx:    cmd,
		}
	}

	argv = make([]reflect.Value, len(cmd.arguments))

	for i := start; i < len(args); i++ {
		v, err := cmd.arguments[i-start](args[i])
		if err != nil {
			return &ErrInvalidUsage{
				Args:   args,
				Prefix: ctx.Prefix,
				Index:  i,
				Err:    err.Error(),
				ctx:    cmd,
			}
		}

		argv[i-start] = v
	}

Call:
	// call the function and parse the error return value
	return callWith(cmd.value, ev, argv...)
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
