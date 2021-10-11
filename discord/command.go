package discord

import (
	"encoding/json"
	"time"
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
	// Options are the parameters for the command.
	//
	// Note that required options must be listed before optional options, and
	// a command, or each individual subcommand, can have a maximum of 25
	// options.
	//
	// It is only present on ChatInputCommands.
	Options []CommandOption `json:"options,omitempty"`
	// NoDefaultPermissions defines whether the command is NOT enabled by
	// default when the app is added to a guild.
	NoDefaultPermission bool `json:"-"`
	// Version is an autoincrementing version identifier updated during
	// substantial record changes
	Version Snowflake `json:"version,omitempty"`
}

// CommandType is the type of the command, which describes the intended
// invokation source of the command.
type CommandType uint

const (
	ChatInputCommand CommandType = iota + 1
	UserCommand
	MessageCommand
)

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
	type RawCommand Command

	// TODO: fix
	type CommandOption struct {
		Type        CommandOptionType     `json:"type"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Required    bool                  `json:"required"`
		Choices     []CommandOptionChoice `json:"choices,omitempty"`
		Options     []CommandOption       `json:"options,omitempty"`

		// If this option is a channel type, the channels shown will be restricted to these types
		ChannelTypes []ChannelType `json:"-"`
	}

	cmd := struct {
		*RawCommand
		DefaultPermission bool `json:"default_permission"`
	}{RawCommand: (*RawCommand)(c)}
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

// CreatedAt returns a time object representing when the command was created.
func (c Command) CreatedAt() time.Time {
	return c.ID.Time()
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
)

// CommandOptionMeta contains the common fields of a CommandOption.
type CommandOptionMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Meta returns itself.
func (m CommandOptionMeta) Meta() CommandOptionMeta { return m }

// CommandOption is a union of command option types. The constructors for
// CommandOption will hint the types that can be a CommandOption.
type CommandOption interface {
	Meta() CommandOptionMeta
	_typ() CommandOptionType
	_cmd()
}

// SubcommandGroupOption is a subcommand group that fits into a CommandOption.
type SubcommandGroupOption struct {
	CommandOptionMeta
	Subcommands []SubcommandOption
}

func NewSubcommandGroupOption(opt SubcommandGroupOption) CommandOption {
	return opt
}

func (s SubcommandGroupOption) _typ() CommandOptionType { return SubcommandGroupOptionType }
func (s SubcommandGroupOption) _cmd()                   {}

// SubcommandOption is a subcommand option that fits into a CommandOption.
type SubcommandOption struct {
	CommandOptionMeta
	Options []CommandOptionValue
}

// NewSubcommandOption creates a new CommandOption from a SubcommandOption.
func NewSubcommandOption(opt SubcommandOption) CommandOption {
	return opt
}

// Meta implements CommandOption.
func (s SubcommandOption) _typ() CommandOptionType { return SubcommandOptionType }
func (s SubcommandOption) _cmd()                   {}

// CommandOptionValue is a subcommand option that fits into a subcommand.
type CommandOptionValue interface {
	Meta() CommandOptionMeta
	_typ() CommandOptionType
	_val()
}

// StringOptionValue is a subcommand option that fits into a CommandOptionValue.
type StringOptionValue struct {
	CommandOptionMeta
	Choices [][2]string
}

// NewStringOptionValue creates a new CommandOptionValue from a
// StringOptionValue.
func NewStringOptionValue(val StringOptionValue) CommandOptionValue {
	return val
}

func (s StringOptionValue) _typ() CommandOptionType { return StringOptionType }
func (s StringOptionValue) _val()                   {}

type IntegerOptionValue struct {
	CommandOptionMeta
	Choices []IntegerChoice
}

func (i IntegerOptionValue) _typ() CommandOptionType { return IntegerOptionType }
func (i IntegerOptionValue) _val()                   {}

// IntegerChoice is a pair of string key to an integer.
type IntegerChoice struct {
	Name  string
	Value int
}

type BooleanOptionValue struct {
	CommandOptionMeta
	Choices []BooleanChoice
}

func (b BooleanOptionValue) _typ() CommandOptionType { return BooleanOptionType }
func (b BooleanOptionValue) _val()                   {}

// BooleanChoice is a pair of string key to a boolean.
type BooleanChoice struct {
	Name  string
	Value bool
}

type UserOptionValue struct {
	CommandOptionMeta
	Choices []UserChoice
}

func (u UserOptionValue) _typ() CommandOptionType { return UserOptionType }
func (u UserOptionValue) _val()                   {}

// UserChoice is a pair of string key to a user ID.
type UserChoice struct {
	Name  string
	Value UserID
}

type ChannelOptionValue struct {
	CommandOptionMeta
	Choices []ChannelChoice
}

func (c ChannelOptionValue) _typ() CommandOptionType { return ChannelOptionType }
func (c ChannelOptionValue) _val()                   {}

// ChannelChoice is a pair of string key to a channel ID.
type ChannelChoice struct {
	Name  string
	Value ChannelID
}

type RoleOptionValue struct {
	CommandOptionMeta
	Choices []RoleChoice
}

func (r RoleOptionValue) _typ() CommandOptionType { return RoleOptionType }
func (r RoleOptionValue) _val()                   {}

// RoleChoice is a pair of string key to a role ID.
type RoleChoice struct {
	Name  string
	Value RoleID
}

type MentionableOptionValue struct {
	CommandOptionMeta
	Choices []MentionableChoice
}

func (m MentionableOptionValue) _typ() CommandOptionType { return MentionableOptionType }
func (m MentionableOptionValue) _val()                   {}

// MentionableChoice is a pair of string key to a mentionable snowflake IDs. To
// use this correctly, use the Resolved field.
type MentionableChoice struct {
	Name  string
	Value Snowflake
}

type NumberOptionValue struct {
	CommandOptionMeta
	Choices []NumberChoice
}

func (n NumberOptionValue) _typ() CommandOptionType { return NumberOptionType }
func (n NumberOptionValue) _val()                   {}

// NumberChoice is a pair of string key to a float64 values.
type NumberChoice struct {
	Name  string
	Value float64
}

/*
type CommandOption struct {
	Type        CommandOptionType     `json:"type"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Required    bool                  `json:"required"`
	Choices     []CommandOptionChoice `json:"choices,omitempty"`
	Options     []CommandOption       `json:"options,omitempty"`

	// If this option is a channel type, the channels shown will be restricted to these types
	ChannelTypes []ChannelType `json:"-"`
}

func (c CommandOption) MarshalJSON() ([]byte, error) {
	type RawOption CommandOption
	option := struct {
		RawOption
		ChannelTypes []uint16 `json:"channel_types,omitempty"`
	}{RawOption: RawOption(c)}

	// []uint8 is marshalled as a base64 string, so we marshal a []uint64 instead.
	if len(c.ChannelTypes) > 0 {
		option.ChannelTypes = make([]uint16, 0, len(c.ChannelTypes))
		for _, t := range c.ChannelTypes {
			option.ChannelTypanes = append(option.ChannelTypes, uint16(t))
		}
	}

	return json.Marshal(option)
}

func (c *CommandOption) UnmarshalJSON(data []byte) error {
	type RawOption CommandOption
	cmd := struct {
		*RawOption
		ChannelTypes []uint16 `json:"channel_types,omitempty"`
	}{RawOption: (*RawOption)(c)}
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	c.ChannelTypes = make([]ChannelType, 0, len(cmd.ChannelTypes))
	for _, t := range cmd.ChannelTypes {
		c.ChannelTypes = append(c.ChannelTypes, ChannelType(t))
	}

	return nil
}
*/

type CommandOptionChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
