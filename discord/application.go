package discord

import (
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

// Command is the base "command" model that belongs to an application. This is
// what you are creating when you POST a new command.
//
// https://discord.com/developers/docs/interactions/slash-commands#application-command-object
type Command struct {
	// ID is the unique id of the command.
	ID CommandID `json:"id"`
	// Type is the type of command.
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
}

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

type CommandOption struct {
	Type        CommandOptionType     `json:"type"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Required    bool                  `json:"required"`
	Choices     []CommandOptionChoice `json:"choices,omitempty"`
	Options     []CommandOption       `json:"options,omitempty"`
}

type CommandOptionType uint

const (
	SubcommandOption CommandOptionType = iota + 1
	SubcommandGroupOption
	StringOption
	IntegerOption
	BooleanOption
	UserOption
	ChannelOption
	RoleOption
	MentionableOption
	NumberOption
)

type CommandOptionChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// https://discord.com/developers/docs/interactions/slash-commands#application-command-permissions-object-guild-application-command-permissions-structure
type GuildCommandPermissions struct {
	ID          CommandID            `json:"id"`
	AppID       AppID                `json:"application_id"`
	GuildID     GuildID              `json:"guild_id"`
	Permissions []CommandPermissions `json:"permissions"`
}

// https://discord.com/developers/docs/interactions/slash-commands#application-command-permissions-object-application-command-permissions-structure
type CommandPermissions struct {
	ID         Snowflake             `json:"id"`
	Type       CommandPermissionType `json:"type"`
	Permission bool                  `json:"permission"`
}

type CommandPermissionType uint8

// https://discord.com/developers/docs/interactions/slash-commands#application-command-permissions-object-application-command-permission-type
const (
	RoleCommandPermission = iota + 1
	UserCommandPermission
)
