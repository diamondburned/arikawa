package gateway

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
)

type (
	// ReadyEvent is the struct for a READY event.
	ReadyEvent struct {
		Version int `json:"version"`

		User      discord.User `json:"user"`
		SessionID string       `json:"session_id"`

		PrivateChannels []discord.Channel  `json:"private_channels"`
		Guilds          []GuildCreateEvent `json:"guilds"`

		Shard *Shard `json:"shard,omitempty"`

		// Undocumented fields

		ReadStates        *ReadStates            `json:"read_state,omitempty"`
		UserSettings      *UserSettings          `json:"user_settings,omitempty"`
		UserGuildSettings *UserGuildSettings     `json:"user_guild_settings,omitempty"`
		Relationships     []discord.Relationship `json:"relationships,omitempty"`
		MergedMembers     [][]SupplementalMember `json:"merged_members,omitempty"`

		FriendSuggestionCount int      `json:"friend_suggestion_count,omitempty"`
		GeoOrderedRTCRegions  []string `json:"geo_ordered_rtc_regions,omitempty"`
	}

	// ReadStates is a struct that contains read state entries. It is an
	// undocumented Discord v8 API.
	ReadStates struct {
		Version int         `json:"version"`
		Partial bool        `json:"partial"`
		Entries []ReadState `json:"entries"`
	}

	// ReadState is a single ReadState entry.
	ReadState struct {
		ChannelID        discord.ChannelID `json:"id"`
		LastMessageID    discord.MessageID `json:"last_message_id"`
		LastPinTimestamp discord.Timestamp `json:"last_pin_timestamp"`
		MentionCount     int               `json:"mention_count"`
	}

	// UserSettings is the struct for (almost) all user settings. It is
	// undocumented.
	UserSettings struct {
		ShowCurrentGame         bool  `json:"show_current_game"`
		DefaultGuildsRestricted bool  `json:"default_guilds_restricted"`
		InlineAttachmentMedia   bool  `json:"inline_attachment_media"`
		InlineEmbedMedia        bool  `json:"inline_embed_media"`
		GIFAutoPlay             bool  `json:"gif_auto_play"`
		RenderEmbeds            bool  `json:"render_embeds"`
		RenderReactions         bool  `json:"render_reactions"`
		AnimateEmoji            bool  `json:"animate_emoji"`
		AnimateStickers         int   `json:"animate_stickers"`
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

		FriendSourceFlags FriendSourceFlags `json:"friend_source_flags"`

		Status       Status            `json:"status"`
		CustomStatus *CustomUserStatus `json:"custom_status"`
	}

	// CustomUserStatus is the custom user status that allows setting an emoji
	// and a piece of text on each user.
	CustomUserStatus struct {
		Text      string            `json:"text"`
		ExpiresAt discord.Timestamp `json:"expires_at,omitempty"`
		EmojiID   discord.EmojiID   `json:"emoji_id,string"`
		EmojiName string            `json:"emoji_name"`
	}

	// UserGuildSettings stores the settings for all user guilds. It is
	// undocumented.
	UserGuildSettings struct {
		Version int                     `json:"version"`
		Partial bool                    `json:"partial"`
		Entries []UserGuildSettingEntry `json:"entries"`
	}

	// UserGuildSettingEntry stores the settings for a single guild. It is
	// undocumented.
	UserGuildSettingEntry struct {
		Version int `json:"version"`

		GuildID discord.GuildID `json:"guild_id"`

		SuppressRoles    bool           `json:"suppress_roles"`
		SuppressEveryone bool           `json:"suppress_everyone"`
		Muted            bool           `json:"muted"`
		MuteConfig       *UserGuildMute `json:"mute_config"`

		MobilePush           bool             `json:"mobile_push"`
		MessageNotifications UserNotification `json:"message_notifications"`

		ChannelOverrides []UserChannelOverride `json:"channel_overrides"`
	}

	// UserGuildMute seems to describe the mute settings of a guild. It belongs
	// to the UserGuildSettingEntry struct and is undocumented.
	UserGuildMute struct {
		SelectedTimeWindow int               `json:"selected_time_window"`
		EndTime            discord.Timestamp `json:"end_time"`
	}

	// A UserChannelOverride struct describes a channel settings override for a
	// users guild settings.
	UserChannelOverride struct {
		Muted bool `json:"muted"`

		MessageNotifications UserNotification  `json:"message_notifications"`
		ChannelID            discord.ChannelID `json:"channel_id"`
	}

	// GuildFolder holds a single folder that you see in the left guild panel.
	GuildFolder struct {
		Name     string            `json:"name"`
		ID       GuildFolderID     `json:"id"`
		GuildIDs []discord.GuildID `json:"guild_ids"`
		Color    discord.Color     `json:"color"`
	}

	// SupplementalMember is the struct for a member in the MergedMembers field
	// of ReadySupplementalEvent. It has slight differences to discord.Member.
	SupplementalMember struct {
		UserID  discord.UserID   `json:"user_id"`
		RoleIDs []discord.RoleID `json:"roles"`

		JoinedAt    discord.Timestamp `json:"joined_at"`
		HoistedRole discord.RoleID    `json:"hoisted_role"`

		Mute bool `json:"mute"`
		Deaf bool `json:"deaf"`

		Nick         string            `json:"nick,omitempty"`
		GuildID      discord.GuildID   `json:"guild_id,omitempty"`
		IsPending    bool              `json:"is_pending,omitempty"`
		PremiumSince discord.Timestamp `json:"premium_since,omitempty"`
	}

	// FriendSourceFlags describes sources that friend requests could be sent
	// from. It belongs to the UserSettings struct and is undocumented.
	FriendSourceFlags struct {
		All           bool `json:"all,omitempty"`
		MutualGuilds  bool `json:"mutual_guilds,omitempty"`
		MutualFriends bool `json:"mutual_friends,omitempty"`
	}
)

// UserNotification is the notification setting for a channel or guild.
type UserNotification uint8

const (
	AllNotifications UserNotification = iota
	OnlyMentions
	NoNotifications
	GuildDefaults
)

// GuildFolderID is possibly a snowflake. It can also be 0 (null) or a low
// number of unknown significance.
type GuildFolderID int64

func (g *GuildFolderID) UnmarshalJSON(b []byte) error {
	var body = string(b)
	if body == "null" {
		return nil
	}

	body = strings.Trim(body, `"`)

	u, err := strconv.ParseInt(body, 10, 64)
	if err != nil {
		return err
	}

	*g = GuildFolderID(u)
	return nil
}

func (g GuildFolderID) MarshalJSON() ([]byte, error) {
	if g == 0 {
		return []byte("null"), nil
	}

	return []byte(strconv.FormatInt(int64(g), 10)), nil
}

// ReadySupplemental event structs.
type (
	// ReadySupplementalEvent is the struct for a READY_SUPPLEMENTAL event,
	// which is an undocumented event.
	ReadySupplementalEvent struct {
		Guilds          []discord.Guild        `json:"guilds"` // only have ID and VoiceStates
		MergedMembers   [][]SupplementalMember `json:"merged_members"`
		MergedPresences MergedPresences        `json:"merged_presences"`
	}

	// MergedPresences is the struct for presences of guilds' members and
	// friends.  It is undocumented.
	MergedPresences struct {
		Guilds  [][]SupplementalPresence `json:"guilds"`
		Friends []SupplementalPresence   `json:"friends"`
	}

	// SupplementalPresence is a single presence for either a guild member or
	// friend. It is used in MergedPresences and is undocumented.
	SupplementalPresence struct {
		UserID discord.UserID `json:"user_id"`

		// Presence contains the rest of this struct.
		Presence

		// LastModified is only present in Friends.
		LastModified discord.UnixMsTimestamp `json:"last_modified,omitempty"`
	}
)
