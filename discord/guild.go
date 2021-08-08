package discord

import "time"

// https://discord.com/developers/docs/resources/guild#guild-object
type Guild struct {
	// ID is the guild id.
	ID GuildID `json:"id"`
	// Name is the guild name (2-100 characters, excluding trailing and leading
	// whitespace).
	Name string `json:"name"`
	// Icon is the icon hash.&nullableUint64{}
	Icon Hash `json:"icon"`
	// Splash is the splash hash.
	Splash Hash `json:"splash,omitempty"`
	// DiscoverySplash is the discovery splash hash.
	//
	// Only present for guilds with the "DISCOVERABLE" feature.
	DiscoverySplash Hash `json:"discovery_splash,omitempty"`

	// Owner is true if the user is the owner of the guild.
	Owner bool `json:"owner,omitempty"`
	// Widget is true if the server widget is enabled.
	Widget bool `json:"widget_enabled,omitempty"`

	// SystemChannelFlags are the system channel flags.
	SystemChannelFlags SystemChannelFlags `json:"system_channel_flags"`
	// Verification is the verification level required for the guild.
	Verification Verification `json:"verification_level"`
	// Notification is the default message notifications level.
	Notification Notification `json:"default_message_notifications"`
	// ExplicitFilter is the explicit content filter level.
	ExplicitFilter ExplicitFilter `json:"explicit_content_filter"`
	// NitroBoost is the premium tier (Server Boost level).
	NitroBoost NitroBoost `json:"premium_tier"`
	// MFA is the required MFA level for the guild.
	MFA MFALevel `json:"mfa"`

	// OwnerID is the id of owner.
	OwnerID UserID `json:"owner_id"`
	// WidgetChannelID is the channel id that the widget will generate an
	// invite to, or null if set to no invite.
	WidgetChannelID ChannelID `json:"widget_channel_id,omitempty"`
	// SystemChannelID is the the id of the channel where guild notices such as
	// welcome messages and boost events are posted.
	SystemChannelID ChannelID `json:"system_channel_id,omitempty"`

	// Permissions are the total permissions for the user in the guild
	// (excludes overrides).
	Permissions Permissions `json:"permissions,string,omitempty"`

	// VoiceRegion is the voice region id for the guild.
	VoiceRegion string `json:"region"`

	// AFKChannelID is the id of the afk channel.
	AFKChannelID ChannelID `json:"afk_channel_id,omitempty"`
	// AFKTimeout is the afk timeout in seconds.
	AFKTimeout Seconds `json:"afk_timeout"`

	// Roles are the roles in the guild.
	Roles []Role `json:"roles"`
	// Emojis are the custom guild emojis.
	Emojis []Emoji `json:"emojis"`
	// Features are the enabled guild features.
	Features []GuildFeature `json:"guild_features"`

	// AppID is the application id of the guild creator if it is bot-created.
	//
	// This field is nullable.
	AppID AppID `json:"application_id,omitempty"`

	// RulesChannelID is the id of the channel where guilds with the "PUBLIC"
	// feature can display rules and/or guidelines.
	RulesChannelID ChannelID `json:"rules_channel_id"`

	// MaxPresences is the maximum number of presences for the guild (the
	// default value, currently 25000, is in effect when null is returned, so
	// effectively when this field is 0).
	MaxPresences uint64 `json:"max_presences,omitempty"`
	// MaxMembers the maximum number of members for the guild.
	MaxMembers uint64 `json:"max_members,omitempty"`

	// VanityURL is the the vanity url code for the guild.
	VanityURLCode string `json:"vanity_url_code,omitempty"`
	// Description is the description for the guild, if the guild is
	// discoverable.
	Description string `json:"description,omitempty"`

	// Banner is the banner hash.
	Banner Hash `json:"banner,omitempty"`

	// NitroBoosters is the number of boosts this guild currently has.
	NitroBoosters uint64 `json:"premium_subscription_count,omitempty"`

	// PreferredLocale is the the preferred locale of a guild with the "PUBLIC"
	// feature; used in server discovery and notices from Discord. Defaults to
	// "en-US".
	PreferredLocale string `json:"preferred_locale"`

	// PublicUpdatesChannelID is the id of the channel where admins and
	// moderators of guilds with the "PUBLIC" feature receive notices from
	// Discord.
	PublicUpdatesChannelID ChannelID `json:"public_updates_channel_id"`

	// MaxVideoChannelUsers is the maximum amount of users in a video channel.
	MaxVideoChannelUsers uint64 `json:"max_video_channel_users,omitempty"`

	// ApproximateMembers is the approximate number of members in this guild,
	// returned by the GuildWithCount method.
	ApproximateMembers uint64 `json:"approximate_member_count,omitempty"`
	// ApproximatePresences is the approximate number of non-offline members in
	// this guild, returned by the GuildWithCount method.
	ApproximatePresences uint64 `json:"approximate_presence_count,omitempty"`
}

// CreatedAt returns a time object representing when the guild was created.
func (g Guild) CreatedAt() time.Time {
	return g.ID.Time()
}

// IconURL returns the URL to the guild icon and auto detects a suitable type.
// An empty string is returned if there's no icon.
func (g Guild) IconURL() string {
	return g.IconURLWithType(AutoImage)
}

// IconURLWithType returns the URL to the guild icon using the passed
// ImageType. An empty string is returned if there's no icon.
//
// Supported ImageTypes: PNG, JPEG, WebP, GIF
func (g Guild) IconURLWithType(t ImageType) string {
	if g.Icon == "" {
		return ""
	}

	return "https://cdn.discordapp.com/icons/" + g.ID.String() + "/" + t.format(g.Icon)
}

// BannerURL returns the URL to the banner, which is the image on top of the
// channels list. This will always return a link to a PNG file.
func (g Guild) BannerURL() string {
	return g.BannerURLWithType(PNGImage)
}

// BannerURLWithType returns the URL to the banner, which is the image on top
// of the channels list using the passed image type.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (g Guild) BannerURLWithType(t ImageType) string {
	if g.Banner == "" {
		return ""
	}

	return "https://cdn.discordapp.com/banners/" +
		g.ID.String() + "/" + t.format(g.Banner)
}

// SplashURL returns the URL to the guild splash, which is the invite page's
// background. This will always return a link to a PNG file.
func (g Guild) SplashURL() string {
	return g.SplashURLWithType(PNGImage)
}

// SplashURLWithType returns the URL to the guild splash, which is the invite
// page's background, using the passed ImageType.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (g Guild) SplashURLWithType(t ImageType) string {
	if g.Splash == "" {
		return ""
	}

	return "https://cdn.discordapp.com/splashes/" +
		g.ID.String() + "/" + t.format(g.Splash)
}

// DiscoverySplashURL returns the URL to the guild discovery splash.
// This will always return a link to a PNG file.
func (g Guild) DiscoverySplashURL() string {
	return g.DiscoverySplashURLWithType(PNGImage)
}

// DiscoverySplashURLWithType returns the URL to the guild discovery splash,
// using the passed ImageType.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (g Guild) DiscoverySplashURLWithType(t ImageType) string {
	if g.DiscoverySplash == "" {
		return ""
	}

	return "https://cdn.discordapp.com/splashes/" +
		g.ID.String() + "/" + t.format(g.DiscoverySplash)
}

// https://discord.com/developers/docs/resources/guild#guild-preview-object
type GuildPreview struct {
	// ID is the guild id.
	ID GuildID `json:"id"`
	// Name is the guild name (2-100 characters).
	Name string `json:"name"`

	// Icon is the icon hash.
	Icon Hash `json:"icon"`
	// Splash is the splash hash.
	Splash Hash `json:"splash"`
	// DiscoverySplash is the discovery splash hash.
	DiscoverySplash Hash `json:"discovery_splash"`

	// Emojis are the custom guild emojis.
	Emojis []Emoji `json:"emojis"`
	// Features are the enabled guild features.
	Features []GuildFeature `json:"guild_features"`

	// ApproximateMembers is the approximate number of members in this guild.
	ApproximateMembers uint64 `json:"approximate_member_count"`
	// ApproximatePresences is the approximate number of online members in this
	// guild.
	ApproximatePresences uint64 `json:"approximate_presence_count"`

	// Description is the description for the guild.
	Description string `json:"description,omitempty"`
}

// CreatedAt returns a time object representing when the guild the preview
// represents was created.
func (g GuildPreview) CreatedAt() time.Time {
	return g.ID.Time()
}

// IconURL returns the URL to the guild icon and auto detects a suitable type.
// An empty string is returned if there's no icon.
func (g GuildPreview) IconURL() string {
	return g.IconURLWithType(AutoImage)
}

// IconURLWithType returns the URL to the guild icon using the passed
// ImageType. An empty string is returned if there's no icon.
//
// Supported ImageTypes: PNG, JPEG, WebP, GIF
func (g GuildPreview) IconURLWithType(t ImageType) string {
	if g.Icon == "" {
		return ""
	}

	return "https://cdn.discordapp.com/icons/" + g.ID.String() + "/" + t.format(g.Icon)
}

// SplashURL returns the URL to the guild splash, which is the invite page's
// background. This will always return a link to a PNG file.
func (g GuildPreview) SplashURL() string {
	return g.SplashURLWithType(PNGImage)
}

// SplashURLWithType returns the URL to the guild splash, which is the invite
// page's background, using the passed ImageType.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (g GuildPreview) SplashURLWithType(t ImageType) string {
	if g.Splash == "" {
		return ""
	}

	return "https://cdn.discordapp.com/splashes/" +
		g.ID.String() + "/" + t.format(g.Splash)
}

// DiscoverySplashURL returns the URL to the guild discovery splash.
// This will always return a link to a PNG file.
func (g GuildPreview) DiscoverySplashURL() string {
	return g.DiscoverySplashURLWithType(PNGImage)
}

// DiscoverySplashURLWithType returns the URL to the guild discovery splash,
// using the passed ImageType.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (g GuildPreview) DiscoverySplashURLWithType(t ImageType) string {
	if g.DiscoverySplash == "" {
		return ""
	}

	return "https://cdn.discordapp.com/splashes/" +
		g.ID.String() + "/" + t.format(g.DiscoverySplash)
}

// https://discord.com/developers/docs/topics/permissions#role-object
type Role struct {
	// ID is the role id.
	ID RoleID `json:"id"`
	// Name is the role name.
	Name string `json:"name"`

	// Permissions is the permission bit set.
	Permissions Permissions `json:"permissions,string"`

	// Position is the position of this role.
	Position int `json:"position"`
	// Color is the integer representation of hexadecimal color code.
	Color Color `json:"color"`

	// Hoist specifies if this role is pinned in the user listing.
	Hoist bool `json:"hoist"`
	// Manages specifies whether this role is managed by an integration.
	Managed bool `json:"managed"`
	// Mentionable specifies whether this role is mentionable.
	Mentionable bool `json:"mentionable"`
}

// CreatedAt returns a time object representing when the role was created.
func (r Role) CreatedAt() time.Time {
	return r.ID.Time()
}

// Mention returns the mention of the Role.
func (r Role) Mention() string {
	return r.ID.Mention()
}

// https://discord.com/developers/docs/resources/guild#guild-member-object
//
// The field user won't be included in the member object attached to
// MESSAGE_CREATE and MESSAGE_UPDATE gateway events.
type Member struct {
	// User is the user this guild member represents.
	User User `json:"user"`
	// Nick is this users guild nickname.
	Nick string `json:"nick,omitempty"`
	// RoleIDs is an array of role object ids.
	RoleIDs []RoleID `json:"roles"`

	// Joined specifies when the user joined the guild.
	Joined Timestamp `json:"joined_at"`
	// BoostedSince specifies when the user started boosting the guild.
	BoostedSince Timestamp `json:"premium_since,omitempty"`

	// Deaf specifies whether the user is deafened in voice channels.
	Deaf bool `json:"deaf"`
	// Mute specifies whether the user is muted in voice channels.
	Mute bool `json:"mute"`

	// IsPending specifies whether the user has not yet passed the guild's Membership Screening requirements
	IsPending bool `json:"pending"`
}

// Mention returns the mention of the role.
func (m Member) Mention() string {
	return "<@!" + m.User.ID.String() + ">"
}

// https://discord.com/developers/docs/resources/guild#ban-object
type Ban struct {
	// Reason is the reason for the ban.
	Reason string `json:"reason,omitempty"`
	// User is the banned user.
	User User `json:"user"`
}

// https://discord.com/developers/docs/resources/guild#integration-object
type Integration struct {
	// ID is the integration id.
	ID IntegrationID `json:"id"`
	// Name is the integration name.
	Name string `json:"name"`
	// Type is the integration type (twitch, youtube, discord).
	Type Service `json:"type"`

	// Enables specifies if the integration is enabled.
	Enabled bool `json:"enabled"`
	// Syncing specifies if the integration is syncing.
	// This field is not provided for bot integrations.
	Syncing bool `json:"syncing,omitempty"`

	// RoleID is the id that this integration uses for "subscribers".
	// This field is not provided for bot integrations.
	RoleID RoleID `json:"role_id,omitempty"`

	// EnableEmoticons specifies whether emoticons should be synced for this
	// integration (twitch only currently).
	// This field is not provided for bot integrations.
	EnableEmoticons bool `json:"enable_emoticons,omitempty"`

	// ExpireBehavior is the behavior of expiring subscribers.
	// This field is not provided for bot integrations.
	ExpireBehavior ExpireBehavior `json:"expire_behavior,omitempty"`
	// ExpireGracePeriod is the grace period (in days) before expiring
	// subscribers.
	// This field is not provided for bot integrations.
	ExpireGracePeriod int `json:"expire_grace_period,omitempty"`

	// User is the user for this integration.
	// This field is not provided for bot integrations.
	User User `json:"user,omitempty"`
	// Account is the integration account information.
	Account IntegrationAccount `json:"account"`

	// SyncedAt specifies when this integration was last synced.
	// This field is not provided for bot integrations.
	SyncedAt Timestamp `json:"synced_at,omitempty"`
	// SubscriberCount specifies how many subscribers the integration has.
	// This field is not provided for bot integrations.
	SubscriberCount int `json:"subscriber_count,omitempty"`
	// Revoked specifies whether the integration has been revoked.
	// This field is not provided for bot integrations.
	Revoked bool `json:"revoked,omitempty"`
	// Application is the bot/OAuth2 application for integrations.
	Application *IntegrationApplication `json:"application,omitempty"`
}

// CreatedAt returns a time object representing when the integration was created.
func (i Integration) CreatedAt() time.Time {
	return i.ID.Time()
}

// https://discord.com/developers/docs/resources/guild#integration-account-object
type IntegrationAccount struct {
	// ID is the id of the account.
	ID string `json:"id"`
	// Name is the name of the account.
	Name string `json:"name"`
}

// https://discord.com/developers/docs/resources/guild#integration-application-object
type IntegrationApplication struct {
	// ID is the id of the app.
	ID IntegrationID `json:"id"`
	// Name is the name of the app.
	Name string `json:"name"`
	// Icon is the icon hash of the app.
	Icon *Hash `json:"icon"`
	// Description is the description of the app.
	Description string `json:"description"`
	// Summary is a summary of the app.
	Summary string `json:"summary"`
	// Bot is the bot associated with the app.
	Bot User `json:"bot,omitempty"`
}

// CreatedAt returns a time object representing when the integration application
// was created.
func (i IntegrationApplication) CreatedAt() time.Time {
	return i.ID.Time()
}

// https://discord.com/developers/docs/resources/guild#get-guild-widget-example-get-guild-widget
type GuildWidget struct {
	// ID is the ID of the guild.
	ID GuildID `json:"id"`
	// Name is the name of the guild.
	Name string `json:"name"`
	// InviteURl is the url of an instant invite to the guild.
	InviteURL string    `json:"instant_invite"`
	Channels  []Channel `json:"channels"`
	Members   []User    `json:"members"`
	// Presence count is the amount of presences in the guild
	PresenceCount int `json:"presence_count"`
}

// https://discord.com/developers/docs/resources/guild#guild-widget-object
type GuildWidgetSettings struct {
	// Enabled specifies whether the widget is enabled.
	Enabled bool `json:"enabled"`
	// ChannelID is the widget channel id.
	ChannelID ChannelID `json:"channel_id,omitempty"`
}

// DefaultMemberColor is the color used for members without colored roles.
var DefaultMemberColor Color = 0x0

// MemberColor computes the effective color of the Member, taking into account
// the role colors.
func MemberColor(guild Guild, member Member) Color {
	c := DefaultMemberColor
	var pos int

	for _, r := range guild.Roles {
		for _, mr := range member.RoleIDs {
			if mr != r.ID {
				continue
			}

			if r.Color > 0 && r.Position > pos {
				c = r.Color
				pos = r.Position
			}
		}
	}

	return c
}

// Presence represents a partial Presence structure used by other structs to be
// easily embedded. It does not contain any ID to identify who it belongs
// to. For more information, refer to the PresenceUpdateEvent struct.
type Presence struct {
	// User is the user presence is being updated for. Only the ID field is
	// guaranteed to be valid per Discord documentation.
	User User `json:"user"`
	// GuildID is the id of the guild
	GuildID GuildID `json:"guild_id"`
	// Status is either "idle", "dnd", "online", or "offline".
	Status Status `json:"status"`
	// Activities are the user's current activities.
	Activities []Activity `json:"activities"`
	// ClientStatus is the user's platform-dependent status.
	ClientStatus ClientStatus `json:"client_status"`
}

type ClientStatus struct {
	// Desktop is the user's status set for an active desktop (Windows,
	// Linux, Mac) application session.
	Desktop Status `json:"desktop,omitempty"`
	// Mobile is the user's status set for an active mobile (iOS, Android)
	// application session.
	Mobile Status `json:"mobile,omitempty"`
	// Web is the user's status set for an active web (browser, bot
	// account) application session.
	Web Status `json:"web,omitempty"`
}
