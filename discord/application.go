package discord

import (
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

type Application struct {
	// ID is the ID of the app.
	ID AppID `json:"id"`
	// Name is the name of the app.
	Name string `json:"string"`
	// Icon is the icon hash of the app.
	Icon *Hash `json:"icon"`
	// Description is the description of the app.
	Description string `json:"string"`
	// RPCOrigins is the RPC origin urls, if RPC is enabled.
	RPCOrigins []string `json:"rpc_origins"`
	// BotPublic is whether users besides the app owner can join the app's bot
	// to guilds.
	BotPublic bool `json:"bot_public"`
	// BotRequiredCodeGrant is whether the app's bot will only join upon
	// completion of the full oauth2 code grant flow.
	BotRequireCodeGrant bool `json:"bot_require_code_grant"`
	// TermsOfServiceURL is the url of the app's terms of service.
	TermsOfServiceURL string `json:"terms_of_service_url"`
	// PrivacyPolicyURL is the url of the app's privacy policy.
	PrivacyPolicyURL string `json:"privacy_policy_url"`
	// Owner is a partial user object containing info on the owner of the
	// application.
	Owner *User `json:"owner"`
	// VerifyKey is the hex encoded key for verification in interactions and
	// the GameSDK's GetTicket.
	VerifyKey string `json:"verify_key"`
	// Team is the team that the application belongs to, if it belongs to one.
	Team *Team `json:"team"`
	// CoverImage the application's default rich presence invite cover image
	// hash.
	CoverImage *Hash `json:"cover_image"`
	// Flags is the application's public flags.
	Flags ApplicationFlags `json:"flags"`

	// The following fields are only present on applications that are games
	// sold on Discord.

	// Summary is the summary field for the store page of the game's primary
	// SKU.
	Summary string `json:"summary"`
	// GuildID is the guild to which the game has been linked.
	GuildID GuildID `json:"guild_ID"`
	// PrimarySKUID is the ID of the "Game SKU" that is created, if it exists.
	PrimarySKUID Snowflake `json:"primary_sku_id"`
	// Slug is the URL slug that links to the game's store page.
	Slug string `json:"slug"`
}

type ApplicationFlags uint32

const (
	AppFlagGatewayPresence ApplicationFlags = 1 << (iota + 12)
	AppFlagGatewayPresenceLimited
	AppFlagGatewayGuildMembers
	AppFlagGatewayGuildMembersLimited
	AppFlagVerificationPendingGuildLimit
	AppFlagEmbedded
)

type Team struct {
	// Icon is a hash of the image of the team's icon.
	Icon *Hash `json:"hash"`
	// ID is the unique ID of the team.
	ID TeamID `json:"id"`
	// Members is the members of the team.
	Members []TeamMember `json:"members"`
	// Name is the name of the team.
	Name string `json:"name"`
	// OwnerUserID is the user ID of the current team owner.
	OwnerID UserID `json:"owner_user_id"`
}

type TeamMember struct {
	// MembershipState is the user's membership state on the team.
	MembershipState MembershipState `json:"membership_state"`
	// Permissions will always be {"*"}
	Permissions []string `json:"permissions"`
	// TeamID is the ID of the parent team of which they are a member.
	TeamID TeamID `json:"team_id"`
	// User is the avatar, discriminator, ID, and username of the user.
	User User `json:"user"`
}

type MembershipState uint8

const (
	MembershipInvited MembershipState = iota + 1
	MembershipAccepted
)

// Command is the base "command" model that belongs to an application. This is
// what you are creating when you POST a new command.
//
// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
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
	// Version is an autoincrementing version identifier updated during
	// substantial record changes
	Version Snowflake `json:"version,omitempty"`
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
			option.ChannelTypes = append(option.ChannelTypes, uint16(t))
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
