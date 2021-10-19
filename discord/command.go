package discord

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

// CommandType is the type of the command, which describes the intended
// invokation source of the command.
type CommandType uint

const (
	ChatInputCommand CommandType = iota + 1
	UserCommand
	MessageCommand
)

// Command is the base "command" model that belongs to an application. This is
// what you are creating when you POST a new command.
//
// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	// ID is the unique id of the command.
	ID CommandID `json:"id"`
	// Type is the intended source of the command.
	Type CommandType `json:"type,omitempty"`
	// AppID is the unique id of the parent application.
	AppID AppID `json:"application_id"`
	// GuildID is the guild id of the command, if not global.
	GuildID GuildID `json:"guild_id,omitempty"`
	// Name is the 1-32 lowercase character name matching ^[\w-]{1,32}$.
	Name string `json:"name"`
	// Description is the 1-100 character description.
	Description string `json:"description"`
	// Options are the parameters for the command. Its types are value types,
	// which can either be a SubcommandOption or a SubcommandGroupOption.
	//
	// Note that required options must be listed before optional options, and
	// a command, or each individual subcommand, can have a maximum of 25
	// options.
	//
	// It is only present on ChatInputCommands.
	Options CommandOptions `json:"options,omitempty"`
	// NoDefaultPermissions defines whether the command is NOT enabled by
	// default when the app is added to a guild.
	NoDefaultPermission bool `json:"-"`
	// Version is an autoincrementing version identifier updated during
	// substantial record changes
	Version Snowflake `json:"version,omitempty"`
}

// CreatedAt returns a time object representing when the command was created.
func (c Command) CreatedAt() time.Time {
	return c.ID.Time()
}

func (c Command) MarshalJSON() ([]byte, error) {
	type RawCommand Command
	cmd := struct {
		RawCommand
		DefaultPermission bool `json:"default_permission"`
	}{RawCommand: RawCommand(c)}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	cmd.DefaultPermission = !c.NoDefaultPermission

	return json.Marshal(cmd)
}

func (c *Command) UnmarshalJSON(data []byte) error {
	type rawCommand Command

	cmd := struct {
		*rawCommand
		DefaultPermission bool `json:"default_permission"`
	}{
		rawCommand: (*rawCommand)(c),
	}

	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	c.NoDefaultPermission = !cmd.DefaultPermission

	// Discord defaults type to 1 if omitted.
	if c.Type == 0 {
		c.Type = ChatInputCommand
	}

	return nil
}

var errUnexpectedOptions = errors.New(
	"unexpected .Options in non-Subcommand and non-SubcommandGroup data",
)

var errUnexpectedChoices = errors.New(
	"unexpected .Choices in Subcommand/SubcommandGroup data",
)

// commandTypeCheckError is returned if a one of Command's Options fails the
// type check.
type commandTypeCheckError struct {
	name   string
	got    interface{}
	expect string
}

// Name returns the name of the erroneous command.
func (err commandTypeCheckError) Name() string {
	return err.name
}

// Data returns the erroneous data that belongs to this error. It is usually
// either a CommandOption or a CommandOptionValue.
func (err commandTypeCheckError) Data() interface{} {
	return err.got
}

// Error implements error.
func (err commandTypeCheckError) Error() string {
	return fmt.Sprintf(
		"error at option name %q: expected %s, got %T",
		err.name, err.expect, err.got,
	)
}

// CommandOptions is used primarily for unmarshaling.
type CommandOptions []CommandOption

// UnmarshalJSON unmarshals b into these CommandOptions.
func (c *CommandOptions) UnmarshalJSON(b []byte) error {
	var unknowns []UnknownCommandOption
	if err := json.Unmarshal(b, &unknowns); err != nil {
		return err
	}

	if len(unknowns) == 0 {
		*c = nil
		return nil
	}

	(*c) = make([]CommandOption, len(unknowns))

	for i, v := range unknowns {
		co, ok := v.data.(CommandOption)
		if !ok {
			return commandTypeCheckError{v.Name, v.data, "CommandOption"}
		}
		(*c)[i] = co
	}

	return nil
}

// UnknownCommandOption is used for unknown or unmarshaled CommandOption values.
// It is used in the unmarshaling stage for all CommandOption types.
//
// An UnknownCommandOption will satisfy both CommandOption and
// CommandOptionValue. Code that type-switches on either of them should not
// assume that only the expected types are used.
type UnknownCommandOption struct {
	CommandOptionMeta

	// Scrap fields used only for unmarshaling:

	// Subcommand or SubcommandGroup only.
	Options []UnknownCommandOption `json:"options,omitempty"`
	// Anything not Subcommand or SubcommandGroup only.
	Choices      json.Raw      `json:"choices,omitempty"`
	ChannelTypes []ChannelType `json:"channel_types,omitempty"`

	raw  json.Raw
	data interface {
		Meta() CommandOptionMeta
		Type() CommandOptionType
	}
}

// Meta returns the CommandOptionMeta for this UnknownCommandOption.
func (u UnknownCommandOption) Meta() CommandOptionMeta {
	return u.CommandOptionMeta
}

// Type returns the supposed type for this UnknownCommandOption.
func (u UnknownCommandOption) Type() CommandOptionType {
	return u.CommandOptionMeta.Type
}

// Raw returns the raw JSON of this UnknownCommandOption. It will only return a
// non-nil blob of JSON if the command option's type cannot be found. If this
// method doesn't return nil, then Data's type will be UnknownCommandOption.
func (u UnknownCommandOption) Raw() json.Raw {
	return u.raw
}

// Data returns the underlying data type, which is a type that satisfies either
// CommandOption or CommandOptionValue.
func (u UnknownCommandOption) Data() interface {
	Meta() CommandOptionMeta
	Type() CommandOptionType
} {
	return u.data
}

// Implement both CommandOption and CommandOptionValue.
func (u UnknownCommandOption) _cmd() {}
func (u UnknownCommandOption) _val() {}

// UnmarshalJSON parses the JSON into the struct as-is then reads all its
// children Options/Choices (if subcommand(group)). Typed command options are
// created into u.Data, or u.Raw if the type is unknown. This is done from the
// bottom up.
func (u *UnknownCommandOption) UnmarshalJSON(b []byte) error {
	type raw UnknownCommandOption

	if err := json.Unmarshal(b, (*raw)(u)); err != nil {
		return errors.Wrap(err, "failed to unmarshal unknown")
	}

	var err error

	switch u.Type() {
	case SubcommandOptionType:
		options := make([]CommandOptionValue, len(u.Options))
		for i, opt := range u.Options {
			ov, ok := opt.data.(CommandOptionValue)
			if !ok {
				return commandTypeCheckError{u.Name, u.data, "CommandOptionValue"}
			}
			options[i] = ov
		}
		u.data = SubcommandOption{
			CommandOptionMeta: u.Meta(),
			Options:           options,
		}
	case SubcommandGroupOptionType:
		options := make([]SubcommandOption, len(u.Options))
		for i, opt := range u.Options {
			ov, ok := opt.data.(SubcommandOption)
			if !ok {
				return commandTypeCheckError{u.Name, u.data, "SubcommandOption"}
			}
			options[i] = ov
		}
		u.data = SubcommandGroupOption{
			CommandOptionMeta: u.Meta(),
			Subcommands:       options,
		}
	case StringOptionType:
		v := StringOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case IntegerOptionType:
		v := IntegerOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case BooleanOptionType:
		v := BooleanOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case UserOptionType:
		v := UserOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case ChannelOptionType:
		v := ChannelOptionValue{CommandOptionMeta: u.Meta(), ChannelTypes: u.ChannelTypes}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case RoleOptionType:
		v := RoleOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case MentionableOptionType:
		v := MentionableOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	case NumberOptionType:
		v := NumberOptionValue{CommandOptionMeta: u.Meta()}
		err = u.unmarshalChoices(&v.Choices)
		u.data = v
	default:
		// Copy the blob of bytes into a new slice.
		u.raw = append(json.Raw(nil), b...)
		u.data = *u
	}

	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal type %d", u.Type())
	}

	return nil
}

func (u *UnknownCommandOption) unmarshalChoices(choices interface{}) error {
	if len(u.Choices) == 0 {
		return nil
	}
	return json.Unmarshal(u.Choices, choices)
}

// CommandOptionType is the enumerated integer type for command options. The
// user usually won't have to touch any of these enum constants.
type CommandOptionType uint

const (
	SubcommandOptionType CommandOptionType = iota + 1
	SubcommandGroupOptionType
	StringOptionType
	IntegerOptionType
	BooleanOptionType
	UserOptionType
	ChannelOptionType
	RoleOptionType
	MentionableOptionType
	NumberOptionType
	maxOptionType // for bound checking
)

// CommandOptionMeta contains the common fields of a CommandOption.
type CommandOptionMeta struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        CommandOptionType `json:"type"`
	Required    bool              `json:"required"`
}

// Meta returns itself.
func (m CommandOptionMeta) Meta() CommandOptionMeta { return m }

// CommandOption is a union of command option types. The constructors for
// CommandOption will hint the types that can be a CommandOption.
type CommandOption interface {
	Meta() CommandOptionMeta
	Type() CommandOptionType
	_cmd()
}

// SubcommandGroupOption is a subcommand group that fits into a CommandOption.
type SubcommandGroupOption struct {
	CommandOptionMeta
	Subcommands []SubcommandOption `json:"options"`
}

// NewSubcommandGroupOption creates a new CommandOption from a
// SubcommandGroupOption.
func NewSubcommandGroupOption(opt SubcommandGroupOption) CommandOption {
	return opt
}

// Type implements CommandOption.
func (s SubcommandGroupOption) Type() CommandOptionType { return SubcommandGroupOptionType }
func (s SubcommandGroupOption) _cmd()                   {}

// SubcommandOption is a subcommand option that fits into a CommandOption.
type SubcommandOption struct {
	CommandOptionMeta
	Options []CommandOptionValue `json:"options"`
}

// NewSubcommandOption creates a new CommandOption from a SubcommandOption.
func NewSubcommandOption(opt SubcommandOption) CommandOption {
	return opt
}

// Type implements CommandOption.
func (s SubcommandOption) Type() CommandOptionType { return SubcommandOptionType }
func (s SubcommandOption) _cmd()                   {}

// CommandOptionValue is a subcommand option that fits into a subcommand.
type CommandOptionValue interface {
	Meta() CommandOptionMeta
	Type() CommandOptionType
	_val()
}

// StringOptionValue is a subcommand option that fits into a CommandOptionValue.
type StringOptionValue struct {
	CommandOptionMeta
	Choices [][2]string `json:"-"`
}

// NewStringOptionValue creates a new CommandOptionValue from a
// StringOptionValue.
func NewStringOptionValue(val StringOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (s StringOptionValue) Type() CommandOptionType { return StringOptionType }
func (s StringOptionValue) _val()                   {}

// StringChoice is a pair of string key to a string.
type StringChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IntegerOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type IntegerOptionValue struct {
	CommandOptionMeta
	Choices []IntegerChoice `json:"choices,omitempty"`
}

// NewIntegerOptionValue creates a new CommandOptionValue from a
// IntegerOptionValue.
func NewIntegerOptionValue(val StringOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (i IntegerOptionValue) Type() CommandOptionType { return IntegerOptionType }
func (i IntegerOptionValue) _val()                   {}

// IntegerChoice is a pair of string key to an integer.
type IntegerChoice struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// BooleanOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type BooleanOptionValue struct {
	CommandOptionMeta
	Choices []BooleanChoice `json:"choices,omitempty"`
}

// NewBooleanOptionValue creates a new CommandOptionValue from a
// BooleanOptionValue.
func NewBooleanOptionValue(val BooleanOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (b BooleanOptionValue) Type() CommandOptionType { return BooleanOptionType }
func (b BooleanOptionValue) _val()                   {}

// BooleanChoice is a pair of string key to a boolean.
type BooleanChoice struct {
	Name  string `json:"name"`
	Value bool   `json:"value"`
}

// UserOptionValue is a subcommand option that fits into a CommandOptionValue.
type UserOptionValue struct {
	CommandOptionMeta
	Choices []UserChoice `json:"choices,omitempty"`
}

// NewUserOptionValue creates a new CommandOptionValue from a UserOptionValue.
func NewUserOptionValue(val UserOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (u UserOptionValue) Type() CommandOptionType { return UserOptionType }
func (u UserOptionValue) _val()                   {}

// UserChoice is a pair of string key to a user ID.
type UserChoice struct {
	Name  string `json:"name"`
	Value UserID `json:"value,string"`
}

// ChannelOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type ChannelOptionValue struct {
	CommandOptionMeta
	Choices      []ChannelChoice `json:"choices,omitempty"`
	ChannelTypes []ChannelType   `json:"channel_types,omitempty"`
}

// NewChannelOptionValue creates a new CommandOptionValue from a
// ChannelOptionValue.
func NewChannelOptionValue(val ChannelOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (c ChannelOptionValue) Type() CommandOptionType { return ChannelOptionType }
func (c ChannelOptionValue) _val()                   {}

// ChannelChoice is a pair of string key to a channel ID.
type ChannelChoice struct {
	Name  string    `json:"name"`
	Value ChannelID `json:"value,string"`
}

// RoleOptionValue is a subcommand option that fits into a CommandOptionValue.
type RoleOptionValue struct {
	CommandOptionMeta
	Choices []RoleChoice `json:"choices,omitempty"`
}

// NewRoleOptionValue creates a new CommandOptionValue from a RoleOptionValue.
func NewRoleOptionValue(val RoleOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (r RoleOptionValue) Type() CommandOptionType { return RoleOptionType }
func (r RoleOptionValue) _val()                   {}

// RoleChoice is a pair of string key to a role ID.
type RoleChoice struct {
	Name  string `json:"name"`
	Value RoleID `json:"value,string"`
}

// MentionableOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type MentionableOptionValue struct {
	CommandOptionMeta
	Choices []MentionableChoice `json:"choices,omitempty"`
}

// NewMentionableOptionValue creates a new CommandOptionValue from a
// MentionableOptionValue.
func NewMentionableOptionValue(val MentionableOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (m MentionableOptionValue) Type() CommandOptionType { return MentionableOptionType }
func (m MentionableOptionValue) _val()                   {}

// MentionableChoice is a pair of string key to a mentionable snowflake IDs. To
// use this correctly, use the Resolved field.
type MentionableChoice struct {
	Name  string    `json:"name"`
	Value Snowflake `json:"value,string"`
}

// NumberOptionValue is a subcommand option that fits into a CommandOptionValue.
type NumberOptionValue struct {
	CommandOptionMeta
	Choices []NumberChoice `json:"choices,omitempty"`
}

// NewNumberOptionValue creates a new CommandOptionValue from a
// NumberOptionValue.
func NewNumberOptionValue(val NumberOptionValue) CommandOptionValue {
	return val
}

// Type implements CommandOptionValue.
func (n NumberOptionValue) Type() CommandOptionType { return NumberOptionType }
func (n NumberOptionValue) _val()                   {}

// NumberChoice is a pair of string key to a float64 values.
type NumberChoice struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
