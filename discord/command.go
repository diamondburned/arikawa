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
func (c *Command) CreatedAt() time.Time {
	return c.ID.Time()
}

func (c *Command) MarshalJSON() ([]byte, error) {
	type RawCommand Command
	cmd := struct {
		*RawCommand
		DefaultPermission bool `json:"default_permission"`
	}{RawCommand: (*RawCommand)(c)}

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

	*c = make([]CommandOption, len(unknowns))
	for i, v := range unknowns {
		(*c)[i] = v.data
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
	OptionName string            `json:"name"`
	OptionType CommandOptionType `json:"type"`

	raw  json.Raw
	data CommandOption
}

// Name returns the supposeed name for this UnknownCommandOption.
func (u *UnknownCommandOption) Name() string {
	return u.OptionName
}

// Type returns the supposed type for this UnknownCommandOption.
func (u *UnknownCommandOption) Type() CommandOptionType {
	return u.OptionType
}

// Raw returns the raw JSON of this UnknownCommandOption. It will only return a
// non-nil blob of JSON if the command option's type cannot be found. If this
// method doesn't return nil, then Data's type will be UnknownCommandOption.
func (u *UnknownCommandOption) Raw() json.Raw {
	return u.raw
}

// Data returns the underlying data type, which is a type that satisfies either
// CommandOption or CommandOptionValue.
func (u *UnknownCommandOption) Data() CommandOption {
	return u.data
}

// Implement both CommandOption and CommandOptionValue.
func (u *UnknownCommandOption) _val() {}

// UnmarshalJSON parses the JSON into the struct as-is then reads all its
// children Options/Choices (if subcommand(group)). Typed command options are
// created into u.Data, or u.Raw if the type is unknown. This is done from the
// bottom up.
func (u *UnknownCommandOption) UnmarshalJSON(b []byte) error {
	type unknown UnknownCommandOption

	if err := json.Unmarshal(b, (*unknown)(u)); err != nil {
		return errors.Wrap(err, "failed to unmarshal unknown")
	}

	switch u.Type() {
	case SubcommandOptionType:
		u.data = &SubcommandOption{}
	case SubcommandGroupOptionType:
		u.data = &SubcommandGroupOption{}
	case StringOptionType:
		u.data = &StringOptionValue{}
	case IntegerOptionType:
		u.data = &IntegerOptionValue{}
	case BooleanOptionType:
		u.data = &BooleanOptionValue{}
	case UserOptionType:
		u.data = &UserOptionValue{}
	case ChannelOptionType:
		u.data = &ChannelOptionValue{}
	case RoleOptionType:
		u.data = &RoleOptionValue{}
	case MentionableOptionType:
		u.data = &MentionableOptionValue{}
	case NumberOptionType:
		u.data = &NumberOptionValue{}
	default:
		// Copy the blob of bytes into a new slice.
		u.raw = append(json.Raw(nil), b...)
		u.data = u
		return nil
	}

	if err := json.Unmarshal(b, u.data); err != nil {
		return errors.Wrapf(err, "failed to unmarshal type %d", u.Type())
	}

	return nil
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

// CommandOption is a union of command option types. The constructors for
// CommandOption will hint the types that can be a CommandOption.
type CommandOption interface {
	Name() string
	Type() CommandOptionType
}

// Maintaining these structs is quite an effort. If a new field is added into
// the generic CommandOption type, you MUST update ALL CommandOption structs.
// This means copy-pasting, yes.

// SubcommandGroupOption is a subcommand group that fits into a CommandOption.
type SubcommandGroupOption struct {
	OptionName  string              `json:"name"`
	Description string              `json:"description"`
	Required    bool                `json:"required"`
	Subcommands []*SubcommandOption `json:"options"`
}

// Name implements CommandOption.
func (s *SubcommandGroupOption) Name() string { return s.OptionName }

// Type implements CommandOption.
func (s *SubcommandGroupOption) Type() CommandOptionType { return SubcommandGroupOptionType }

// SubcommandOption is a subcommand option that fits into a CommandOption.
type SubcommandOption struct {
	OptionName  string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	// Options contains command option values. All CommandOption types except
	// for SubcommandOption and SubcommandGroupOption will implement this
	// interface.
	Options []CommandOptionValue `json:"options"`
}

// Name implements CommandOption.
func (s *SubcommandOption) Name() string { return s.OptionName }

// Type implements CommandOption.
func (s *SubcommandOption) Type() CommandOptionType { return SubcommandOptionType }

// UnmarshalJSON unmarshals the given JSON bytes. It actually does
// type-checking.
func (s *SubcommandOption) UnmarshalJSON(b []byte) error {
	type raw SubcommandOption

	var opt struct {
		*raw
		Type    CommandOptionType      `json:"type"`
		Options []UnknownCommandOption `json:"options"`
	}

	opt.raw = (*raw)(s)

	if err := json.Unmarshal(b, &opt); err != nil {
		return err
	}

	if opt.Type != SubcommandOptionType {
		return fmt.Errorf("unexpected (not SubcommandOption) type %d", s.Type())
	}

	s.Options = make([]CommandOptionValue, len(opt.Options))
	for i, opt := range opt.Options {
		ov, ok := opt.data.(CommandOptionValue)
		if !ok {
			return commandTypeCheckError{opt.OptionName, opt.data, "CommandOptionValue"}
		}
		s.Options[i] = ov
	}

	return nil
}

// CommandOptionValue is a subcommand option that fits into a subcommand.
type CommandOptionValue interface {
	CommandOption
	_val()
}

// StringOptionValue is a subcommand option that fits into a CommandOptionValue.
type StringOptionValue struct {
	OptionName  string         `json:"name"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Choices     []StringChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (s *StringOptionValue) Name() string { return s.OptionName }

// Type implements CommandOptionValue.
func (s *StringOptionValue) Type() CommandOptionType { return StringOptionType }
func (s *StringOptionValue) _val()                   {}

// StringChoice is a pair of string key to a string.
type StringChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IntegerOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type IntegerOptionValue struct {
	OptionName  string          `json:"name"`
	Description string          `json:"description"`
	Required    bool            `json:"required"`
	Choices     []IntegerChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (i *IntegerOptionValue) Name() string { return i.OptionName }

// Type implements CommandOptionValue.
func (i *IntegerOptionValue) Type() CommandOptionType { return IntegerOptionType }
func (i *IntegerOptionValue) _val()                   {}

// IntegerChoice is a pair of string key to an integer.
type IntegerChoice struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// BooleanOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type BooleanOptionValue struct {
	OptionName  string          `json:"name"`
	Description string          `json:"description"`
	Required    bool            `json:"required"`
	Choices     []BooleanChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (b *BooleanOptionValue) Name() string { return b.OptionName }

// Type implements CommandOptionValue.
func (b *BooleanOptionValue) Type() CommandOptionType { return BooleanOptionType }
func (b *BooleanOptionValue) _val()                   {}

// BooleanChoice is a pair of string key to a boolean.
type BooleanChoice struct {
	Name  string `json:"name"`
	Value bool   `json:"value"`
}

// UserOptionValue is a subcommand option that fits into a CommandOptionValue.
type UserOptionValue struct {
	OptionName  string       `json:"name"`
	Description string       `json:"description"`
	Required    bool         `json:"required"`
	Choices     []UserChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (u *UserOptionValue) Name() string { return u.OptionName }

// Type implements CommandOptionValue.
func (u *UserOptionValue) Type() CommandOptionType { return UserOptionType }
func (u *UserOptionValue) _val()                   {}

// UserChoice is a pair of string key to a user ID.
type UserChoice struct {
	Name  string `json:"name"`
	Value UserID `json:"value,string"`
}

// ChannelOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type ChannelOptionValue struct {
	OptionName   string          `json:"name"`
	Description  string          `json:"description"`
	Required     bool            `json:"required"`
	Choices      []ChannelChoice `json:"choices,omitempty"`
	ChannelTypes []ChannelType   `json:"channel_types,omitempty"`
}

// Name implements CommandOption.
func (c *ChannelOptionValue) Name() string { return c.OptionName }

// Type implements CommandOptionValue.
func (c *ChannelOptionValue) Type() CommandOptionType { return ChannelOptionType }
func (c *ChannelOptionValue) _val()                   {}

// ChannelChoice is a pair of string key to a channel ID.
type ChannelChoice struct {
	Name  string    `json:"name"`
	Value ChannelID `json:"value,string"`
}

// RoleOptionValue is a subcommand option that fits into a CommandOptionValue.
type RoleOptionValue struct {
	OptionName  string       `json:"name"`
	Description string       `json:"description"`
	Required    bool         `json:"required"`
	Choices     []RoleChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (r *RoleOptionValue) Name() string { return r.OptionName }

// Type implements CommandOptionValue.
func (r *RoleOptionValue) Type() CommandOptionType { return RoleOptionType }
func (r *RoleOptionValue) _val()                   {}

// RoleChoice is a pair of string key to a role ID.
type RoleChoice struct {
	Name  string `json:"name"`
	Value RoleID `json:"value,string"`
}

// MentionableOptionValue is a subcommand option that fits into a
// CommandOptionValue.
type MentionableOptionValue struct {
	OptionName  string              `json:"name"`
	Description string              `json:"description"`
	Required    bool                `json:"required"`
	Choices     []MentionableChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (m *MentionableOptionValue) Name() string { return m.OptionName }

// Type implements CommandOptionValue.
func (m *MentionableOptionValue) Type() CommandOptionType { return MentionableOptionType }
func (m *MentionableOptionValue) _val()                   {}

// MentionableChoice is a pair of string key to a mentionable snowflake IDs. To
// use this correctly, use the Resolved field.
type MentionableChoice struct {
	Name  string    `json:"name"`
	Value Snowflake `json:"value,string"`
}

// NumberOptionValue is a subcommand option that fits into a CommandOptionValue.
type NumberOptionValue struct {
	OptionName  string         `json:"name"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Choices     []NumberChoice `json:"choices,omitempty"`
}

// Name implements CommandOption.
func (n *NumberOptionValue) Name() string { return n.OptionName }

// Type implements CommandOptionValue.
func (n *NumberOptionValue) Type() CommandOptionType { return NumberOptionType }
func (n *NumberOptionValue) _val()                   {}

// NumberChoice is a pair of string key to a float64 values.
type NumberChoice struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
