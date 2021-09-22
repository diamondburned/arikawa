package gateway

import (
	"github.com/diamondburned/arikawa/v3/discord"
)

// Rules: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent

// https://discord.com/developers/docs/topics/gateway#connecting-and-resuming
type (
	HelloEvent struct {
		HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
	}

	// Ready is too big, so it's moved to ready.go

	ResumedEvent struct{}

	// InvalidSessionEvent indicates if the event is resumable.
	InvalidSessionEvent bool
)

// https://discord.com/developers/docs/topics/gateway#channels
type (
	ChannelCreateEvent struct {
		discord.Channel
	}
	ChannelUpdateEvent struct {
		discord.Channel
	}
	ChannelDeleteEvent struct {
		discord.Channel
	}
	ChannelPinsUpdateEvent struct {
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
		ChannelID discord.ChannelID `json:"channel_id,omitempty"`
		LastPin   discord.Timestamp `json:"timestamp,omitempty"`
	}

	ChannelUnreadUpdateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`

		ChannelUnreadUpdates []struct {
			ID            discord.ChannelID `json:"id"`
			LastMessageID discord.MessageID `json:"last_message_id"`
		}
	}

	// ThreadCreateEvent is sent when a thread is created, relevant to the
	// current user, or when the current user is added to a thread.
	ThreadCreateEvent struct {
		discord.Channel
	}

	// ThreadUpdateEvent is sent when a thread is updated.
	ThreadUpdateEvent struct {
		discord.Channel
	}

	// ThreadDeleteEvent is sent when a thread relevant to the current user is
	// deleted.
	ThreadDeleteEvent struct {
		// ID is the id of this channel.
		ID discord.ChannelID `json:"id"`
		// GuildID is the id of the guild.
		GuildID discord.GuildID `json:"guild_id,omitempty"`
		// Type is the type of channel.
		Type discord.ChannelType `json:"type,omitempty"`
		// ParentID is the id of the text channel this thread was created.
		ParentID discord.ChannelID `json:"parent_id,omitempty"`
	}

	// ThreadListSyncEvent is sent when the current user gains access to a
	// channel.
	ThreadListSyncEvent struct {
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

	// ThreadMemberUpdateEvent is sent when the thread member object for the
	// current user is updated.
	ThreadMemberUpdateEvent struct {
		discord.ThreadMember
	}

	// ThreadMembersUpdateEvent is sent when anyone is added to or removed from
	// a thread. If the current user does not have the GUILD_MEMBERS Gateway
	// Intent, then this event will only be sent if the current user was added
	// to or removed from the thread.
	ThreadMembersUpdateEvent struct {
		// ID is the id of the thread.
		ID discord.ChannelID
		// GuildID is the id of the guild.
		GuildID discord.GuildID
		// MemberCount is the approximate number of members in the thread,
		// capped at 50.
		MemberCount int
		// AddedMembers are the users who were added to the thread.
		AddedMembers []discord.ThreadMember
		// RemovedUserIDs are the ids of the users who were removed from the
		// thread.
		RemovedMemberIDs []discord.UserID
	}
)

// https://discord.com/developers/docs/topics/gateway#guilds
type (
	GuildCreateEvent struct {
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
	GuildUpdateEvent struct {
		discord.Guild
	}
	GuildDeleteEvent struct {
		ID discord.GuildID `json:"id"`
		// Unavailable if false == removed
		Unavailable bool `json:"unavailable"`
	}

	GuildBanAddEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		User    discord.User    `json:"user"`
	}
	GuildBanRemoveEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		User    discord.User    `json:"user"`
	}

	GuildEmojisUpdateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		Emojis  []discord.Emoji `json:"emojis"`
	}

	GuildIntegrationsUpdateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
	}

	GuildMemberAddEvent struct {
		discord.Member
		GuildID discord.GuildID `json:"guild_id"`
	}
	GuildMemberRemoveEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		User    discord.User    `json:"user"`
	}
	GuildMemberUpdateEvent struct {
		GuildID discord.GuildID  `json:"guild_id"`
		RoleIDs []discord.RoleID `json:"roles"`
		User    discord.User     `json:"user"`
		Nick    string           `json:"nick"`
	}

	// GuildMembersChunkEvent is sent when Guild Request Members is called.
	GuildMembersChunkEvent struct {
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

	// GuildMemberListUpdate is an undocumented event. It's received when the
	// client sends over GuildSubscriptions with the Channels field used.
	// The State package does not handle this event.
	GuildMemberListUpdate struct {
		ID          string          `json:"id"`
		GuildID     discord.GuildID `json:"guild_id"`
		MemberCount uint64          `json:"member_count"`
		OnlineCount uint64          `json:"online_count"`

		// Groups is all the visible role sections.
		Groups []GuildMemberListGroup `json:"groups"`

		Ops []GuildMemberListOp `json:"ops"`
	}
	GuildMemberListGroup struct {
		ID    string `json:"id"` // either discord.RoleID, "online" or "offline"
		Count uint64 `json:"count"`
	}
	GuildMemberListOp struct {
		// Mysterious string, so far spotted to be [SYNC, INSERT, UPDATE, DELETE].
		Op string `json:"op"`

		// NON-SYNC ONLY
		// Only available for Ops that aren't "SYNC".
		Index int                   `json:"index,omitempty"`
		Item  GuildMemberListOpItem `json:"item,omitempty"`

		// SYNC ONLY
		// Range requested in GuildSubscribeData.
		Range [2]int `json:"range,omitempty"`
		// Items is basically a linear list of roles and members, similarly to
		// how the client renders it. No, it's not nested.
		Items []GuildMemberListOpItem `json:"items,omitempty"`
	}
	// GuildMemberListOpItem is an enum. Either of the fields are provided, but
	// never both. Refer to (*GuildMemberListUpdate).Ops for more.
	GuildMemberListOpItem struct {
		Group  *GuildMemberListGroup `json:"group,omitempty"`
		Member *struct {
			discord.Member
			HoistedRole string           `json:"hoisted_role"`
			Presence    discord.Presence `json:"presence"`
		} `json:"member,omitempty"`
	}

	GuildRoleCreateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		Role    discord.Role    `json:"role"`
	}
	GuildRoleUpdateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		Role    discord.Role    `json:"role"`
	}
	GuildRoleDeleteEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
		RoleID  discord.RoleID  `json:"role_id"`
	}
)

func (u GuildMemberUpdateEvent) Update(m *discord.Member) {
	m.RoleIDs = u.RoleIDs
	m.User = u.User
	m.Nick = u.Nick
}

// https://discord.com/developers/docs/topics/gateway#invites
type (
	InviteCreateEvent struct {
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
	InviteDeleteEvent struct {
		Code      string            `json:"code"`
		ChannelID discord.ChannelID `json:"channel_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
)

// https://discord.com/developers/docs/topics/gateway#messages
type (
	MessageCreateEvent struct {
		discord.Message
		Member *discord.Member `json:"member,omitempty"`
	}
	MessageUpdateEvent struct {
		discord.Message
		Member *discord.Member `json:"member,omitempty"`
	}
	MessageDeleteEvent struct {
		ID        discord.MessageID `json:"id"`
		ChannelID discord.ChannelID `json:"channel_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageDeleteBulkEvent struct {
		IDs       []discord.MessageID `json:"ids"`
		ChannelID discord.ChannelID   `json:"channel_id"`
		GuildID   discord.GuildID     `json:"guild_id,omitempty"`
	}

	MessageReactionAddEvent struct {
		UserID    discord.UserID    `json:"user_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`

		Emoji discord.Emoji `json:"emoji,omitempty"`

		GuildID discord.GuildID `json:"guild_id,omitempty"`
		Member  *discord.Member `json:"member,omitempty"`
	}
	MessageReactionRemoveEvent struct {
		UserID    discord.UserID    `json:"user_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		Emoji     discord.Emoji     `json:"emoji"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveAllEvent struct {
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveEmojiEvent struct {
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		Emoji     discord.Emoji     `json:"emoji"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}

	MessageAckEvent struct {
		MessageID discord.MessageID `json:"message_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
	}
)

// https://discord.com/developers/docs/topics/gateway#presence
type (
	// ClientStatus is the user's platform-dependent status.
	//
	// https://discord.com/developers/docs/topics/gateway#client-status-object

	// PresenceUpdateEvent represents the structure of the Presence Update Event
	// object.
	//
	// https://discord.com/developers/docs/topics/gateway#presence-update-presence-update-event-fields
	PresenceUpdateEvent struct {
		discord.Presence
	}

	PresencesReplaceEvent []PresenceUpdateEvent

	// SessionsReplaceEvent is an undocumented user event. It's likely used for
	// current user's presence updates.
	SessionsReplaceEvent []struct {
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

	TypingStartEvent struct {
		ChannelID discord.ChannelID     `json:"channel_id"`
		UserID    discord.UserID        `json:"user_id"`
		Timestamp discord.UnixTimestamp `json:"timestamp"`

		GuildID discord.GuildID `json:"guild_id,omitempty"`
		Member  *discord.Member `json:"member,omitempty"`
	}

	UserUpdateEvent struct {
		discord.User
	}
)

// https://discord.com/developers/docs/topics/gateway#voice
type (
	VoiceStateUpdateEvent struct {
		discord.VoiceState
	}
	VoiceServerUpdateEvent struct {
		Token    string          `json:"token"`
		GuildID  discord.GuildID `json:"guild_id"`
		Endpoint string          `json:"endpoint"`
	}
)

// https://discord.com/developers/docs/topics/gateway#webhooks
type (
	WebhooksUpdateEvent struct {
		GuildID   discord.GuildID   `json:"guild_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
	}
)

type InteractionCreateEvent struct {
	discord.Interaction
}

// Undocumented
type (
	UserGuildSettingsUpdateEvent struct {
		UserGuildSetting
	}
	UserSettingsUpdateEvent struct {
		UserSettings
	}
	UserNoteUpdateEvent struct {
		ID   discord.UserID `json:"id"`
		Note string         `json:"note"`
	}
)

type (
	RelationshipAddEvent struct {
		discord.Relationship
	}
	RelationshipRemoveEvent struct {
		discord.Relationship
	}
)

type (
	ApplicationCommandUpdateEvent struct {
		discord.Command
		GuildID discord.GuildID `json:"guild_id"`
	}
)
