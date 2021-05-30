package gateway

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
)

type (
	// ReadyEvent is the struct for a READY event.
	ReadyEvent struct {
		Shard *Shard `json:"shard,omitempty"`
		// Undocumented
		UserSettings *UserSettings `json:"user_settings,omitempty"`
		SessionID    string        `json:"session_id"`
		// Undocumented
		GeoOrderedRTCRegions []string           `json:"geo_ordered_rtc_regions,omitempty"`
		Guilds               []GuildCreateEvent `json:"guilds"`
		PrivateChannels      []discord.Channel  `json:"private_channels"`
		// Undocumented
		Presences []Presence `json:"presences,omitempty"`
		// Undocumented
		ReadStates []ReadState `json:"read_state,omitempty"`
		// Undocumented
		UserGuildSettings []UserGuildSetting `json:"user_guild_settings,omitempty"`
		// Undocumented
		Relationships []discord.Relationship `json:"relationships,omitempty"`
		User          discord.User           `json:"user"`
		// Undocumented
		FriendSuggestionCount int `json:"friend_suggestion_count,omitempty"`
		Version               int `json:"version"`
	}

	// ReadState is a single ReadState entry. It is undocumented.
	ReadState struct {
		LastPinTimestamp discord.Timestamp `json:"last_pin_timestamp"`
		ChannelID        discord.ChannelID `json:"id"`
		LastMessageID    discord.MessageID `json:"last_message_id"`
		MentionCount     int               `json:"mention_count"`
	}

	// UserSettings is the struct for (almost) all user settings. It is
	// undocumented.
	UserSettings struct {
		CustomStatus            *CustomUserStatus `json:"custom_status"`
		Status                  Status            `json:"status"`
		Theme                   string            `json:"theme"`
		Locale                  string            `json:"locale"`
		RestrictedGuilds        []discord.GuildID `json:"restricted_guilds"`
		GuildFolders            []GuildFolder     `json:"guild_folders"`
		GuildPositions          []discord.GuildID `json:"guild_positions"`
		AnimateStickers         int               `json:"animate_stickers"`
		TimezoneOffset          int               `json:"timezone_offset"`
		FriendSourceFlags       FriendSourceFlags `json:"friend_source_flags"`
		DeveloperMode           bool              `json:"developer_mode"`
		ConvertEmoticons        bool              `json:"convert_emoticons"`
		ExplicitContentFilter   uint8             `json:"explicit_content_filter"`
		DisableGamesTab         bool              `json:"disable_games_tab"`
		MessageDisplayCompact   bool              `json:"message_display_compact"`
		DetectPlatformAccounts  bool              `json:"detect_platform_accounts"`
		StreamNotification      bool              `json:"stream_notification_enabled"`
		AccessibilityDetection  bool              `json:"allow_accessibility_detection"`
		ContactSync             bool              `json:"contact_sync_enabled"`
		NativePhoneIntegration  bool              `json:"native_phone_integration_enabled"`
		EnableTTSCommand        bool              `json:"enable_tts_command"`
		AnimateEmoji            bool              `json:"animate_emoji"`
		RenderReactions         bool              `json:"render_reactions"`
		RenderEmbeds            bool              `json:"render_embeds"`
		GIFAutoPlay             bool              `json:"gif_auto_play"`
		InlineEmbedMedia        bool              `json:"inline_embed_media"`
		InlineAttachmentMedia   bool              `json:"inline_attachment_media"`
		DefaultGuildsRestricted bool              `json:"default_guilds_restricted"`
		ShowCurrentGame         bool              `json:"show_current_game"`
	}

	// CustomUserStatus is the custom user status that allows setting an emoji
	// and a piece of text on each user.
	CustomUserStatus struct {
		ExpiresAt discord.Timestamp `json:"expires_at,omitempty"`
		Text      string            `json:"text"`
		EmojiName string            `json:"emoji_name"`
		EmojiID   discord.EmojiID   `json:"emoji_id,string"`
	}

	// UserGuildSetting stores the settings for a single guild. It is
	// undocumented.
	UserGuildSetting struct {
		MuteConfig       *UserMuteConfig       `json:"mute_config"`
		ChannelOverrides []UserChannelOverride `json:"channel_overrides"`
		GuildID          discord.GuildID       `json:"guild_id"`
		SuppressEveryone bool                  `json:"suppress_everyone"`
		Muted            bool                  `json:"muted"`
		MobilePush       bool                  `json:"mobile_push"`
		Notifications    UserNotification      `json:"message_notifications"`
		SuppressRoles    bool                  `json:"suppress_roles"`
	}

	// A UserChannelOverride struct describes a channel settings override for a
	// users guild settings.
	UserChannelOverride struct {
		MuteConfig    *UserMuteConfig   `json:"mute_config"`
		ChannelID     discord.ChannelID `json:"channel_id"`
		Muted         bool              `json:"muted"`
		Notifications UserNotification  `json:"message_notifications"`
	}

	// UserMuteConfig seems to describe the mute settings. It belongs to the
	// UserGuildSettingEntry and UserChannelOverride structs and is
	// undocumented.
	UserMuteConfig struct {
		EndTime            discord.Timestamp `json:"end_time"`
		SelectedTimeWindow int               `json:"selected_time_window"`
	}

	// GuildFolder holds a single folder that you see in the left guild panel.
	GuildFolder struct {
		Name     string            `json:"name"`
		GuildIDs []discord.GuildID `json:"guild_ids"`
		ID       GuildFolderID     `json:"id"`
		Color    discord.Color     `json:"color"`
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
	body := string(b)
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

// ReadySupplemental event structs. For now, this event is never used, and its
// usage have yet been discovered.
type (
	// ReadySupplementalEvent is the struct for a READY_SUPPLEMENTAL event,
	// which is an undocumented event.
	ReadySupplementalEvent struct {
		Guilds          []GuildCreateEvent     `json:"guilds"` // only have ID and VoiceStates
		MergedMembers   [][]SupplementalMember `json:"merged_members"`
		MergedPresences MergedPresences        `json:"merged_presences"`
	}

	// SupplementalMember is the struct for a member in the MergedMembers field
	// of ReadySupplementalEvent. It has slight differences to discord.Member.
	SupplementalMember struct {
		// Joined specifies when the user joined the guild.
		Joined discord.Timestamp `json:"joined_at"`
		// BoostedSince specifies when the user started boosting the guild.
		BoostedSince discord.Timestamp `json:"premium_since,omitempty"`
		Nick         string            `json:"nick,omitempty"`
		RoleIDs      []discord.RoleID  `json:"roles"`
		UserID       discord.UserID    `json:"user_id"`
		HoistedRole  discord.RoleID    `json:"hoisted_role"`
		GuildID      discord.GuildID   `json:"guild_id,omitempty"`
		IsPending    bool              `json:"is_pending,omitempty"`
		Mute         bool              `json:"mute"`
		Deaf         bool              `json:"deaf"`
	}

	// MergedPresences is the struct for presences of guilds' members and
	// friends. It is undocumented.
	MergedPresences struct {
		Guilds  [][]SupplementalPresence `json:"guilds"`
		Friends []SupplementalPresence   `json:"friends"`
	}

	// SupplementalPresence is a single presence for either a guild member or
	// friend. It is used in MergedPresences and is undocumented.
	SupplementalPresence struct {
		// ClientStaus is the user's platform-dependent status.
		ClientStatus ClientStatus `json:"client_status"`
		// Status is either "idle", "dnd", "online", or "offline".
		Status Status `json:"status"`
		// Activities are the user's current activities.
		Activities []discord.Activity `json:"activities"`
		UserID     discord.UserID     `json:"user_id"`
		// LastModified is only present in Friends.
		LastModified discord.UnixMsTimestamp `json:"last_modified,omitempty"`
	}
)

// ConvertSupplementalMember converts a SupplementalMember to a regular Member.
func ConvertSupplementalMember(sm SupplementalMember) discord.Member {
	return discord.Member{
		User:         discord.User{ID: sm.UserID},
		Nick:         sm.Nick,
		RoleIDs:      sm.RoleIDs,
		Joined:       sm.Joined,
		BoostedSince: sm.BoostedSince,
		Deaf:         sm.Deaf,
		Mute:         sm.Mute,
	}
}

// ConvertSupplementalPresence converts a SupplementalPresence to a regular
// Presence with an empty GuildIDs.
func ConvertSupplementalPresence(sp SupplementalPresence) Presence {
	return Presence{
		User:         discord.User{ID: sp.UserID},
		Status:       sp.Status,
		Activities:   sp.Activities,
		ClientStatus: sp.ClientStatus,
	}
}
