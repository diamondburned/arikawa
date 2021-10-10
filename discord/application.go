package discord

type Application struct {
	// ID is the ID of the app.
	ID AppID `json:"id"`
	// Name is the name of the app.
	Name string `json:"name"`
	// Icon is the icon hash of the app.
	Icon *Hash `json:"icon"`
	// Description is the description of the app.
	Description string `json:"description"`
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
