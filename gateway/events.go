package gateway

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

//go:generate go run ../utils/cmd/genevent -o event_methods.go

// Event is a type alias for ws.Event. It exists for convenience and describes
// the same event as any other ws.Event.
type Event ws.Event

// Rule: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent.
// Ready is too big, so it's moved to ready.go.

const (
	dispatchOp            ws.OpCode = 0 // recv
	heartbeatOp           ws.OpCode = 1 // send/recv
	identifyOp            ws.OpCode = 2 // send...
	statusUpdateOp        ws.OpCode = 3 //
	voiceStateUpdateOp    ws.OpCode = 4 //
	voiceServerPingOp     ws.OpCode = 5 //
	resumeOp              ws.OpCode = 6 //
	reconnectOp           ws.OpCode = 7 // recv
	requestGuildMembersOp ws.OpCode = 8 // send
	invalidSessionOp      ws.OpCode = 9 // recv...
	helloOp               ws.OpCode = 10
	heartbeatAckOp        ws.OpCode = 11
	callConnectOp         ws.OpCode = 13
	guildSubscriptionsOp  ws.OpCode = 14
)

// OpUnmarshalers contains the Op unmarshalers for this gateway.
var OpUnmarshalers = ws.NewOpUnmarshalers()

// HeartbeatCommand is a command for Op 1. It is the last sequence number to be
// sent.
type HeartbeatCommand int

// HeartbeatAckEvent is an event for Op 11.
type HeartbeatAckEvent struct{}

// ReconnectEvent is an event for Op 7.
type ReconnectEvent struct{}

// HelloEvent is an event for Op 10.
//
// https://discord.com/developers/docs/topics/gateway#connecting-and-resuming
type HelloEvent struct {
	HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
}

// ResumeCommand is a command for Op 6. It describes the Resume send command.
// This is not to be confused with ResumedEvent, which is an event that Discord
// sends us.
type ResumeCommand struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// InvalidSessionEvent is an event for Op 9. It indicates if the event is
// resumable.
//
// https://discord.com/developers/docs/topics/gateway#connecting-and-resuming
type InvalidSessionEvent bool

// RequestGuildMembersCommand is a command for Op 8.
type RequestGuildMembersCommand struct {
	// GuildIDs contains the ids of the guilds to request data from. Multiple
	// guilds can only be requested when using user accounts.
	GuildIDs []discord.GuildID `json:"guild_id"`
	UserIDs  []discord.UserID  `json:"user_ids,omitempty"`

	Query     string `json:"query,omitempty"`
	Limit     uint   `json:"limit,omitempty"`
	Presences bool   `json:"presences"`
	Nonce     string `json:"nonce,omitempty"`
}

// UpdateVoiceStateCommand is a command for Op 4.
type UpdateVoiceStateCommand struct {
	GuildID   discord.GuildID   `json:"guild_id"`
	ChannelID discord.ChannelID `json:"channel_id"` // nullable
	SelfMute  bool              `json:"self_mute"`
	SelfDeaf  bool              `json:"self_deaf"`
}

// UpdatePresenceCommand is a command for Op 3. It is sent by this client to
// indicate a presence or status update.
type UpdatePresenceCommand struct {
	Since discord.UnixMsTimestamp `json:"since"` // 0 if not idle

	// Activities can be null or an empty slice.
	Activities []discord.Activity `json:"activities"`

	Status discord.Status `json:"status"`
	AFK    bool           `json:"afk"`
}

// GuildSubscribeCommand is a command for Op 14. It is undocumented.
type GuildSubscribeCommand struct {
	Typing     bool            `json:"typing"`
	Threads    bool            `json:"threads"`
	Activities bool            `json:"activities"`
	GuildID    discord.GuildID `json:"guild_id"`

	// Channels is not documented. It's used to fetch the right members sidebar.
	Channels map[discord.ChannelID][][2]int `json:"channels,omitempty"`
}

// ResumedEvent is a dispatch event. It is sent by Discord whenever we've
// successfully caught up to all events after resuming.
type ResumedEvent struct{}

// ChannelCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#channels
type ChannelCreateEvent struct {
	discord.Channel
}

// ChannelUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#channels
type ChannelUpdateEvent struct {
	discord.Channel
}

// ChannelDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#channels
type ChannelDeleteEvent struct {
	discord.Channel
}

// ChannelPinsUpdateEvent is a dispatch event.
type ChannelPinsUpdateEvent struct {
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	ChannelID discord.ChannelID `json:"channel_id,omitempty"`
	LastPin   discord.Timestamp `json:"timestamp,omitempty"`
}

// ChannelUnreadUpdateEvent is a dispatch event.
type ChannelUnreadUpdateEvent struct {
	GuildID discord.GuildID `json:"guild_id"`

	ChannelUnreadUpdates []struct {
		ID            discord.ChannelID `json:"id"`
		LastMessageID discord.MessageID `json:"last_message_id"`
	}
}

// ThreadCreateEvent is a dispatch event. It is sent when a thread is created,
// relevant to the current user, or when the current user is added to a thread.
type ThreadCreateEvent struct {
	discord.Channel
}

// ThreadUpdateEvent is a dispatch event. It is sent when a thread is updated.
type ThreadUpdateEvent struct {
	discord.Channel
}

// ThreadDeleteEvent is a dispatch event. It is sent when a thread relevant to
// the current user is deleted.
type ThreadDeleteEvent struct {
	// ID is the id of this channel.
	ID discord.ChannelID `json:"id"`
	// GuildID is the id of the guild.
	GuildID discord.GuildID `json:"guild_id,omitempty"`
	// Type is the type of channel.
	Type discord.ChannelType `json:"type,omitempty"`
	// ParentID is the id of the text channel this thread was created.
	ParentID discord.ChannelID `json:"parent_id,omitempty"`
}

// ThreadListSyncEvent is a dispatch event. It is sent when the current user
// gains access to a channel.
type ThreadListSyncEvent struct {
	// GuildID is the id of the guild.
	GuildID discord.GuildID `json:"guild_id"`
	// ChannelIDs are the parent channel ids whose threads are being
	// synced. If nil, then threads were synced for the entire guild.
	// This slice may contain ChannelIDs that have no active threads as
	// well, so you know to clear that data.
	ChannelIDs []discord.ChannelID    `json:"channel_ids,omitempty"`
	Threads    []discord.Channel      `json:"threads"`
	Members    []discord.ThreadMember `json:"members"`
}

// ThreadMemberUpdateEvent is a dispatch event. It is sent when the thread
// member object for the current user is updated.
type ThreadMemberUpdateEvent struct {
	discord.ThreadMember
}

// ThreadMembersUpdateEvent is a dispatch event. It is sent when anyone is added
// to or removed from a thread. If the current user does not have the
// GUILD_MEMBERS Gateway Intent, then this event will only be sent if the
// current user was added to or removed from the thread.
type ThreadMembersUpdateEvent struct {
	// ID is the id of the thread.
	ID discord.ChannelID `json:"id"`
	// GuildID is the id of the guild.
	GuildID discord.GuildID `json:"guild_id"`
	// MemberCount is the approximate number of members in the thread,
	// capped at 50.
	MemberCount int `json:"member_count"`
	// AddedMembers are the users who were added to the thread.
	AddedMembers []discord.ThreadMember `json:"added_members,omitempty"`
	// RemovedUserIDs are the ids of the users who were removed from the
	// thread.
	RemovedMemberIDs []discord.UserID `json:"removed_member_ids,omitempty"`
}

// GuildCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildCreateEvent struct {
	discord.Guild

	Joined      discord.Timestamp `json:"joined_at,omitempty"`
	Large       bool              `json:"large,omitempty"`
	Unavailable bool              `json:"unavailable,omitempty"`
	MemberCount uint64            `json:"member_count,omitempty"`

	VoiceStates []discord.VoiceState `json:"voice_states,omitempty"`
	Members     []discord.Member     `json:"members,omitempty"`
	Channels    []discord.Channel    `json:"channels,omitempty"`
	Threads     []discord.Channel    `json:"threads,omitempty"`
	Presences   []discord.Presence   `json:"presences,omitempty"`
}

// GuildUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildUpdateEvent struct {
	discord.Guild
}

// GuildDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildDeleteEvent struct {
	ID discord.GuildID `json:"id"`
	// Unavailable if false == removed
	Unavailable bool `json:"unavailable"`
}

// GuildBanAddEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildBanAddEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	User    discord.User    `json:"user"`
}

// GuildBanRemoveEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildBanRemoveEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	User    discord.User    `json:"user"`
}

// GuildEmojisUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildEmojisUpdateEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	Emojis  []discord.Emoji `json:"emojis"`
}

// GuildIntegrationsUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildIntegrationsUpdateEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
}

// GuildMemberAddEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildMemberAddEvent struct {
	discord.Member
	GuildID discord.GuildID `json:"guild_id"`
}

// GuildMemberRemoveEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildMemberRemoveEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	User    discord.User    `json:"user"`
}

// GuildMemberUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildMemberUpdateEvent struct {
	GuildID                    discord.GuildID   `json:"guild_id"`
	RoleIDs                    []discord.RoleID  `json:"roles"`
	User                       discord.User      `json:"user"`
	Nick                       string            `json:"nick"`
	Avatar                     discord.Hash      `json:"avatar"`
	IsPending                  bool              `json:"pending,omitempty"`
	CommunicationDisabledUntil discord.Timestamp `json:"communication_disabled_until"`
}

// UpdateMember updates the given discord.Member.
func (u *GuildMemberUpdateEvent) UpdateMember(m *discord.Member) {
	m.RoleIDs = u.RoleIDs
	m.User = u.User
	m.Nick = u.Nick
	m.Avatar = u.Avatar
	m.IsPending = u.IsPending
	m.CommunicationDisabledUntil = u.CommunicationDisabledUntil
}

// GuildMembersChunkEvent is a dispatch event. It is sent when the Guild Request
// Members command is sent.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildMembersChunkEvent struct {
	GuildID discord.GuildID  `json:"guild_id"`
	Members []discord.Member `json:"members"`

	ChunkIndex int `json:"chunk_index"`
	ChunkCount int `json:"chunk_count"`

	// Whatever's not found goes here
	NotFound []string `json:"not_found,omitempty"`

	// Only filled if requested
	Presences []discord.Presence `json:"presences,omitempty"`
	Nonce     string             `json:"nonce,omitempty"`
}

// GuildMemberListUpdate is a dispatch event. It is an undocumented event. It's
// received when the client sends over GuildSubscriptions with the Channels
// field used.  The State package does not handle this event.
type GuildMemberListUpdate struct {
	ID          string          `json:"id"`
	GuildID     discord.GuildID `json:"guild_id"`
	MemberCount uint64          `json:"member_count"`
	OnlineCount uint64          `json:"online_count"`

	// Groups is all the visible role sections.
	Groups []GuildMemberListGroup `json:"groups"`

	Ops []GuildMemberListOp `json:"ops"`
}

// GuildMemberListGroup is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildMemberListGroup struct {
	ID    string `json:"id"` // either discord.RoleID, "online" or "offline"
	Count uint64 `json:"count"`
}

// GuildMemberListOp is an entry of every operation in GuildMemberListUpdate.
type GuildMemberListOp struct {
	// Mysterious string, so far spotted to be [SYNC, INSERT, UPDATE, DELETE].
	Op string `json:"op"`

	// NON-SYNC ONLY
	// Only available for Ops that aren't "SYNC".
	Index int                   `json:"index,omitempty"`
	Item  GuildMemberListOpItem `json:"item,omitempty"`

	// SYNC ONLY
	// Range requested in GuildSubscribeCommand.
	Range [2]int `json:"range,omitempty"`
	// Items is basically a linear list of roles and members, similarly to
	// how the client renders it. No, it's not nested.
	Items []GuildMemberListOpItem `json:"items,omitempty"`
}

// GuildMemberListOpItem is a union of either Group or Member. Refer to
// (*GuildMemberListUpdate).Ops for more.
type GuildMemberListOpItem struct {
	Group  *GuildMemberListGroup `json:"group,omitempty"`
	Member *struct {
		discord.Member
		HoistedRole string           `json:"hoisted_role"`
		Presence    discord.Presence `json:"presence"`
	} `json:"member,omitempty"`
}

// GuildRoleCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildRoleCreateEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	Role    discord.Role    `json:"role"`
}

// GuildRoleUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildRoleUpdateEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	Role    discord.Role    `json:"role"`
}

// GuildRoleDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guilds
type GuildRoleDeleteEvent struct {
	GuildID discord.GuildID `json:"guild_id"`
	RoleID  discord.RoleID  `json:"role_id"`
}

// InviteCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#invites
type InviteCreateEvent struct {
	Code      string            `json:"code"`
	CreatedAt discord.Timestamp `json:"created_at"`
	ChannelID discord.ChannelID `json:"channel_id"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`

	// Similar to discord.Invite
	Inviter    *discord.User          `json:"inviter,omitempty"`
	Target     *discord.User          `json:"target_user,omitempty"`
	TargetType discord.InviteUserType `json:"target_user_type,omitempty"`

	discord.InviteMetadata
}

// InviteDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#invites
type InviteDeleteEvent struct {
	Code      string            `json:"code"`
	ChannelID discord.ChannelID `json:"channel_id"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
}

// MessageCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageCreateEvent struct {
	discord.Message
	Member *discord.Member `json:"member,omitempty"`
}

// MessageUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageUpdateEvent struct {
	discord.Message
	Member *discord.Member `json:"member,omitempty"`
}

// MessageDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageDeleteEvent struct {
	ID        discord.MessageID `json:"id"`
	ChannelID discord.ChannelID `json:"channel_id"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
}

// MessageDeleteBulkEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageDeleteBulkEvent struct {
	IDs       []discord.MessageID `json:"ids"`
	ChannelID discord.ChannelID   `json:"channel_id"`
	GuildID   discord.GuildID     `json:"guild_id,omitempty"`
}

// MessageReactionAddEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageReactionAddEvent struct {
	UserID    discord.UserID    `json:"user_id"`
	ChannelID discord.ChannelID `json:"channel_id"`
	MessageID discord.MessageID `json:"message_id"`

	Emoji discord.Emoji `json:"emoji,omitempty"`

	GuildID discord.GuildID `json:"guild_id,omitempty"`
	Member  *discord.Member `json:"member,omitempty"`
}

// MessageReactionRemoveEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageReactionRemoveEvent struct {
	UserID    discord.UserID    `json:"user_id"`
	ChannelID discord.ChannelID `json:"channel_id"`
	MessageID discord.MessageID `json:"message_id"`
	Emoji     discord.Emoji     `json:"emoji"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
}

// MessageReactionRemoveAllEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageReactionRemoveAllEvent struct {
	ChannelID discord.ChannelID `json:"channel_id"`
	MessageID discord.MessageID `json:"message_id"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
}

// MessageReactionRemoveEmojiEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#messages
type MessageReactionRemoveEmojiEvent struct {
	ChannelID discord.ChannelID `json:"channel_id"`
	MessageID discord.MessageID `json:"message_id"`
	Emoji     discord.Emoji     `json:"emoji"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
}

// MessageAckEvent is a dispatch event.
type MessageAckEvent struct {
	MessageID discord.MessageID `json:"message_id"`
	ChannelID discord.ChannelID `json:"channel_id"`
}

// PresenceUpdateEvent is a dispatch event. It represents the structure of the
// Presence Update Event object.
//
// https://discord.com/developers/docs/topics/gateway#presence-update-presence-update-event-fields
type PresenceUpdateEvent struct {
	discord.Presence
}

// PresencesReplaceEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#presence
type PresencesReplaceEvent []PresenceUpdateEvent

// SessionsReplaceEvent is a dispatch event. It is undocumented. It's likely
// used for current user's presence updates.
type SessionsReplaceEvent []struct {
	Status    discord.Status `json:"status"`
	SessionID string         `json:"session_id"`

	Activities []discord.Activity `json:"activities"`

	ClientInfo struct {
		Version int    `json:"version"`
		OS      string `json:"os"`
		Client  string `json:"client"`
	} `json:"client_info"`

	Active bool `json:"active"`
}

// TypingStartEvent is a dispatch event.
type TypingStartEvent struct {
	ChannelID discord.ChannelID     `json:"channel_id"`
	UserID    discord.UserID        `json:"user_id"`
	Timestamp discord.UnixTimestamp `json:"timestamp"`

	GuildID discord.GuildID `json:"guild_id,omitempty"`
	Member  *discord.Member `json:"member,omitempty"`
}

// UserUpdateEvent is a dispatch event.
type UserUpdateEvent struct {
	discord.User
}

// VoiceStateUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#voice
type VoiceStateUpdateEvent struct {
	discord.VoiceState
}

// VoiceServerUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#voice
type VoiceServerUpdateEvent struct {
	Token    string          `json:"token"`
	GuildID  discord.GuildID `json:"guild_id"`
	Endpoint string          `json:"endpoint"`
}

// WebhooksUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#webhooks
type WebhooksUpdateEvent struct {
	GuildID   discord.GuildID   `json:"guild_id"`
	ChannelID discord.ChannelID `json:"channel_id"`
}

// InteractionCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#webhooks
type InteractionCreateEvent struct {
	discord.InteractionEvent
}

// Undocumented

// UserGuildSettingsUpdateEvent is a dispatch event. It is undocumented.
type UserGuildSettingsUpdateEvent struct {
	UserGuildSetting
}

// UserSettingsUpdateEvent is a dispatch event. It is undocumented.
type UserSettingsUpdateEvent struct {
	UserSettings
}

// UserNoteUpdateEvent is a dispatch event. It is undocumented.
type UserNoteUpdateEvent struct {
	ID   discord.UserID `json:"id"`
	Note string         `json:"note"`
}

// RelationshipAddEvent is a dispatch event. It is undocumented.
type RelationshipAddEvent struct {
	discord.Relationship
}

// RelationshipRemoveEvent is a dispatch event. It is undocumented.
type RelationshipRemoveEvent struct {
	discord.Relationship
}

// ReadyEvent is a dispatch event for READY.
//
// https://discord.com/developers/docs/topics/gateway#ready
type ReadyEvent struct {
	Version int `json:"v"`

	User      discord.User `json:"user"`
	SessionID string       `json:"session_id"`

	PrivateChannels []discord.Channel  `json:"private_channels"`
	Guilds          []GuildCreateEvent `json:"guilds"`

	Shard *Shard `json:"shard,omitempty"`

	Application struct {
		ID    discord.AppID            `json:"id"`
		Flags discord.ApplicationFlags `json:"flags"`
	} `json:"application"`

	// Undocumented fields

	UserSettings      *UserSettings          `json:"user_settings,omitempty"`
	ReadStates        []ReadState            `json:"read_state,omitempty"`
	UserGuildSettings []UserGuildSetting     `json:"user_guild_settings,omitempty"`
	Relationships     []discord.Relationship `json:"relationships,omitempty"`
	Presences         []discord.Presence     `json:"presences,omitempty"`

	FriendSuggestionCount int      `json:"friend_suggestion_count,omitempty"`
	GeoOrderedRTCRegions  []string `json:"geo_ordered_rtc_regions,omitempty"`
}

// Ready subtypes.
type (
	// ReadState is a single ReadState entry. It is undocumented.
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

		TimezoneOffset int `json:"timezone_offset"`

		Locale string `json:"locale"`
		Theme  string `json:"theme"`

		GuildPositions   []discord.GuildID `json:"guild_positions"`
		GuildFolders     []GuildFolder     `json:"guild_folders"`
		RestrictedGuilds []discord.GuildID `json:"restricted_guilds"`

		FriendSourceFlags FriendSourceFlags `json:"friend_source_flags"`

		Status       discord.Status    `json:"status"`
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

	// UserGuildSetting stores the settings for a single guild. It is
	// undocumented.
	UserGuildSetting struct {
		GuildID discord.GuildID `json:"guild_id"`

		SuppressRoles    bool            `json:"suppress_roles"`
		SuppressEveryone bool            `json:"suppress_everyone"`
		Muted            bool            `json:"muted"`
		MuteConfig       *UserMuteConfig `json:"mute_config"`

		MobilePush    bool             `json:"mobile_push"`
		Notifications UserNotification `json:"message_notifications"`

		ChannelOverrides []UserChannelOverride `json:"channel_overrides"`
	}

	// A UserChannelOverride struct describes a channel settings override for a
	// users guild settings.
	UserChannelOverride struct {
		Muted         bool              `json:"muted"`
		MuteConfig    *UserMuteConfig   `json:"mute_config"`
		Notifications UserNotification  `json:"message_notifications"`
		ChannelID     discord.ChannelID `json:"channel_id"`
	}

	// UserMuteConfig seems to describe the mute settings. It belongs to the
	// UserGuildSettingEntry and UserChannelOverride structs and is
	// undocumented.
	UserMuteConfig struct {
		SelectedTimeWindow int               `json:"selected_time_window"`
		EndTime            discord.Timestamp `json:"end_time"`
	}

	// GuildFolder holds a single folder that you see in the left guild panel.
	GuildFolder struct {
		Name     string            `json:"name"`
		ID       GuildFolderID     `json:"id"`
		GuildIDs []discord.GuildID `json:"guild_ids"`
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

// ReadySupplementalEvent is a dispatch event for READY_SUPPLEMENTAL. It is an
// undocumented event. For now, this event is never used, and its usage have yet
// been discovered.
type ReadySupplementalEvent struct {
	Guilds          []GuildCreateEvent     `json:"guilds"` // only have ID and VoiceStates
	MergedMembers   [][]SupplementalMember `json:"merged_members"`
	MergedPresences MergedPresences        `json:"merged_presences"`
}

// ReadySupplemental event structs.
type (
	// SupplementalMember is the struct for a member in the MergedMembers field
	// of ReadySupplementalEvent. It has slight differences to discord.Member.
	SupplementalMember struct {
		UserID  discord.UserID   `json:"user_id"`
		Nick    string           `json:"nick,omitempty"`
		RoleIDs []discord.RoleID `json:"roles"`

		GuildID     discord.GuildID `json:"guild_id,omitempty"`
		IsPending   bool            `json:"pending,omitempty"`
		HoistedRole discord.RoleID  `json:"hoisted_role"`

		Mute bool `json:"mute"`
		Deaf bool `json:"deaf"`

		// Joined specifies when the user joined the guild.
		Joined discord.Timestamp `json:"joined_at"`

		// BoostedSince specifies when the user started boosting the guild.
		BoostedSince discord.Timestamp `json:"premium_since,omitempty"`
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
		UserID discord.UserID `json:"user_id"`

		// Status is either "idle", "dnd", "online", or "offline".
		Status discord.Status `json:"status"`
		// Activities are the user's current activities.
		Activities []discord.Activity `json:"activities"`
		// ClientStaus is the user's platform-dependent status.
		ClientStatus discord.ClientStatus `json:"client_status"`

		// LastModified is only present in Friends.
		LastModified discord.UnixMsTimestamp `json:"last_modified,omitempty"`
	}
)

// ConvertSupplementalMembers converts a SupplementalMember to a regular Member.
func ConvertSupplementalMembers(sms []SupplementalMember) []discord.Member {
	members := make([]discord.Member, len(sms))
	for i, sm := range sms {
		members[i] = discord.Member{
			User:         discord.User{ID: sm.UserID},
			Nick:         sm.Nick,
			RoleIDs:      sm.RoleIDs,
			Joined:       sm.Joined,
			BoostedSince: sm.BoostedSince,
			Deaf:         sm.Deaf,
			Mute:         sm.Mute,
			IsPending:    sm.IsPending,
		}
	}
	return members
}

// ConvertSupplementalPresences converts a SupplementalPresence to a regular
// Presence with an empty GuildID.
func ConvertSupplementalPresences(sps []SupplementalPresence) []discord.Presence {
	presences := make([]discord.Presence, len(sps))
	for i, sp := range sps {
		presences[i] = discord.Presence{
			User:         discord.User{ID: sp.UserID},
			Status:       sp.Status,
			Activities:   sp.Activities,
			ClientStatus: sp.ClientStatus,
		}
	}
	return presences
}

// GuildScheduledEventCreateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guild-scheduled-event-create
type GuildScheduledEventCreateEvent struct {
	discord.GuildScheduledEvent
}

// GuildScheduledEventUpdateEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guild-scheduled-event-update
type GuildScheduledEventUpdateEvent struct {
	discord.GuildScheduledEvent
}

// GuildScheduledEventDeleteEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guild-scheduled-event-delete
type GuildScheduledEventDeleteEvent struct {
	discord.GuildScheduledEvent
}

// GuildScheduledEventUserAddEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guild-scheduled-event-user-add
type GuildScheduledEventUserAddEvent struct {
	// EventID is the id of the scheduled event
	EventID discord.EventID `json:"guild_scheduled_event_id"`
	// UserID is the id of the user being added
	UserID discord.UserID `json:"user_id"`
	// GuildID is the id of where the scheduled event belongs
	GuildID discord.GuildID `json:"guild_id"`
}

// GuildScheduledEventUserRemoveEvent is a dispatch event.
//
// https://discord.com/developers/docs/topics/gateway#guild-scheduled-event-user-remove
type GuildScheduledEventUserRemoveEvent struct {
	// EventID is the id of the scheduled event
	EventID discord.EventID `json:"guild_scheduled_event_id"`
	// UserID is the id of the user being removed
	UserID discord.UserID `json:"user_id"`
	// GuildID is the id of where the scheduled event belongs
	GuildID discord.GuildID `json:"guild_id"`
}
