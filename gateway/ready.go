package gateway

import "github.com/diamondburned/arikawa/discord"

type ReadyEvent struct {
	Version int `json:"version"`

	User      discord.User `json:"user"`
	SessionID string       `json:"session_id"`

	PrivateChannels []discord.Channel  `json:"private_channels"`
	Guilds          []GuildCreateEvent `json:"guilds"`

	Shard *Shard `json:"shard"`

	// Undocumented fields
	Settings          *UserSettings       `json:"user_settings,omitempty"`
	UserGuildSettings []UserGuildSettings `json:"user_guild_settings,omitempty"`

	ReadState []ReadState        `json:"read_state,omitempty"`
	Presences []discord.Presence `json:"presences,omitempty"`

	Relationships []discord.Relationship    `json:"relationships,omitempty"`
	Notes         map[discord.UserID]string `json:"notes,omitempty"`
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
	AccessibilityDetection  bool  `json:"allow_accessibility_detection"`
	ContactSync             bool  `json:"contact_sync_enabled"`
	NativePhoneIntegration  bool  `json:"native_phone_integration_enabled"`

	Locale string `json:"locale"`
	Theme  string `json:"theme"`

	GuildPositions   []discord.GuildID `json:"guild_positions"`
	GuildFolders     []GuildFolder     `json:"guild_folders"`
	RestrictedGuilds []discord.GuildID `json:"restricted_guilds"`

	FriendSourceFlags struct {
		All           bool `json:"all"`
		MutualGuilds  bool `json:"mutual_guilds"`
		MutualFriends bool `json:"mutual_friends"`
	} `json:"friend_source_flags"`

	Status       discord.Status `json:"status"`
	CustomStatus struct {
		Text      string            `json:"text"`
		ExpiresAt discord.Timestamp `json:"expires_at,omitempty"`
		EmojiID   discord.EmojiID   `json:"emoji_id,string"`
		EmojiName string            `json:"emoji_name"`
	} `json:"custom_status"`
}

// A UserGuildSettings stores data for a users guild settings.
type UserGuildSettings struct {
	GuildID discord.GuildID `json:"guild_id"`

	SuppressEveryone bool `json:"suppress_everyone"`
	SuppressRoles    bool `json:"suppress_roles"`
	Muted            bool `json:"muted"`
	MobilePush       bool `json:"mobile_push"`

	MessageNotifications UserNotification          `json:"message_notifications"`
	ChannelOverrides     []SettingsChannelOverride `json:"channel_overrides"`
}

// UserNotification is the notification setting for a channel or guild.
type UserNotification uint8

const (
	AllNotifications UserNotification = iota
	OnlyMentions
	NoNotifications
	GuildDefaults
)

type ReadState struct {
	ChannelID     discord.ChannelID `json:"id"`
	LastMessageID discord.MessageID `json:"last_message_id"`
	MentionCount  int               `json:"mention_count"`
}

// A UserGuildSettingsChannelOverride stores data for a channel override for a
// users guild settings.
type SettingsChannelOverride struct {
	Muted bool `json:"muted"`

	MessageNotifications UserNotification  `json:"message_notifications"`
	ChannelID            discord.ChannelID `json:"channel_id"`
}

// GuildFolder holds a single folder that you see in the left guild panel.
type GuildFolder struct {
	Name     string            `json:"name"`
	ID       int64             `json:"id"`
	GuildIDs []discord.GuildID `json:"guild_ids"`
	Color    discord.Color     `json:"color"`
}
