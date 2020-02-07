package gateway

import "github.com/diamondburned/arikawa/discord"

type ReadyEvent struct {
	Version int `json:"version"`

	User      discord.User `json:"user"`
	SessionID string       `json:"session_id"`

	PrivateChannels []discord.Channel `json:"private_channels"`
	Guilds          []discord.Guild   `json:"guilds"`

	Shard *Shard `json:"shard"`

	// Undocumented fields
	Settings          *UserSettings                `json:"user_settings"`
	UserGuildSettings []UserGuildSettings          `json:"user_guild_settings"`
	Relationships     []Relationship               `json:"relationships"`
	Presences         []discord.Presence           `json:"presences,omitempty"`
	Notes             map[discord.Snowflake]string `json:"notes,omitempty"`
}

type UserSettings struct {
	ShowCurrentGame         bool  `json:"show_current_game"`
	DefaultGuildsRestricted bool  `json:"default_guilds_restricted"`
	InlineAttachmentMedia   bool  `json:"inline_attachment_media"`
	InlineEmbedMedia        bool  `json:"inline_embed_media"`
	GIFAutoPlay             bool  `json:"gif_auto_play"`
	RenderEmbeds            bool  `json:"render_embeds"`
	RenderReactions         bool  `json:"render_reactions"`
	AnimateEmoji            bool  `json:"animate_emoji"`
	EnableTTSCommand        bool  `json:"enable_tts_command"`
	MessageDisplayCompact   bool  `json:"message_display_compact"`
	ConvertEmoticons        bool  `json:"convert_emoticons"`
	ExplicitContentFilter   uint8 `json:"explicit_content_filter"` // ???
	DisableGamesTab         bool  `json:"disable_games_tab"`
	DeveloperMode           bool  `json:"developer_mode"`
	DetectPlatformAccounts  bool  `json:"detect_platform_accounts"`
	StreamNotification      bool  `json:"stream_notification_enabled"`
	AccessibilityDetection  bool  `json:"allow_accessbility_detection"`
	ContactSync             bool  `json:"contact_sync_enabled"`
	NativePhoneIntegration  bool  `json:"native_phone_integration_enabled"`

	Locale string `json:"locale"`
	Theme  string `json:"theme"`

	GuildPositions   []discord.Snowflake `json:"guild_positions"`
	GuildFolders     []GuildFolder       `json:"guild_folders"`
	RestrictedGuilds []discord.Snowflake `json:"restricted_guilds"`

	FriendSourceFlags struct {
		All           bool `json:"all"`
		MutualGuilds  bool `json:"mutual_guilds"`
		MutualFriends bool `json:"mutual_friends"`
	} `json:"friend_source_flags"`

	Status       discord.Status `json:"status"`
	CustomStatus struct {
		Text      string            `json:"text"`
		ExpiresAt discord.Timestamp `json:"expires_at,omitempty"`
		EmojiID   discord.Snowflake `json:"emoji_id,string"`
		EmojiName string            `json:"emoji_name"`
	} `json:"custom_status"`
}

// A UserGuildSettingsChannelOverride stores data for a channel override for a
// users guild settings.
type SettingsChannelOverride struct {
	Muted                bool `json:"muted"`
	MessageNotifications int  `json:"message_notifications"` // TODO: document

	ChannelID discord.Snowflake `json:"channel_id"`
}

// A UserGuildSettings stores data for a users guild settings.
type UserGuildSettings struct {
	SupressEveryone      bool `json:"suppress_everyone"`
	Muted                bool `json:"muted"`
	MobilePush           bool `json:"mobile_push"`
	MessageNotifications int  `json:"message_notifications"`

	GuildID          discord.Snowflake         `json:"guild_id"`
	ChannelOverrides []SettingsChannelOverride `json:"channel_overrides"`
}

// GuildFolder holds a single folder that you see in the left guild panel.
type GuildFolder struct {
	Name     string              `json:"name"`
	ID       discord.Snowflake   `json:"id"`
	GuildIDs []discord.Snowflake `json:"guild_ids"`
	Color    discord.Color       `json:"color"`
}

// A Relationship between the logged in user and Relationship.User
type Relationship struct {
	ID   string           `json:"id"`
	User discord.User     `json:"user"`
	Type RelationshipType `json:"type"`
}

type RelationshipType uint8

const (
	_ RelationshipType = iota
	FriendRelationship
	BlockedRelationship
	IncomingFriendRequest
	SentFriendRequest
)
