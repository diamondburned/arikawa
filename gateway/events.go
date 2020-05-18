package gateway

import "github.com/diamondburned/arikawa/discord"

// Rules: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent

// https://discordapp.com/developers/docs/topics/gateway#connecting-and-resuming
type (
	HelloEvent struct {
		HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
	}

	// Ready is too big, so it's moved to ready.go

	ResumedEvent struct{}

	// InvalidSessionEvent indicates if the event is resumable.
	InvalidSessionEvent bool
)

// https://discordapp.com/developers/docs/topics/gateway#channels
type (
	ChannelCreateEvent     discord.Channel
	ChannelUpdateEvent     discord.Channel
	ChannelDeleteEvent     discord.Channel
	ChannelPinsUpdateEvent struct {
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
		ChannelID discord.Snowflake `json:"channel_id,omitempty"`
		LastPin   discord.Timestamp `json:"timestamp,omitempty"`
	}

	ChannelUnreadUpdateEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`

		ChannelUnreadUpdates []struct {
			ID            discord.Snowflake `json:"id"`
			LastMessageID discord.Snowflake `json:"last_message_id"`
		}
	}
)

// https://discordapp.com/developers/docs/topics/gateway#guilds
type (
	GuildCreateEvent struct {
		discord.Guild

		Joined      discord.Timestamp `json:"timestamp,omitempty"`
		Large       bool              `json:"large,omitempty"`
		Unavailable bool              `json:"unavailable,omitempty"`
		MemberCount uint64            `json:"member_count,omitempty"`

		VoiceStates []discord.VoiceState `json:"voice_states,omitempty"`
		Members     []discord.Member     `json:"members,omitempty"`
		Channels    []discord.Channel    `json:"channels,omitempty"`
		Presences   []discord.Presence   `json:"presences,omitempty"`
	}
	GuildUpdateEvent discord.Guild
	GuildDeleteEvent struct {
		ID discord.Snowflake `json:"id"`
		// Unavailable if false == removed
		Unavailable bool `json:"unavailable"`
	}

	GuildBanAddEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		User    discord.User      `json:"user"`
	}
	GuildBanRemoveEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		User    discord.User      `json:"user"`
	}

	GuildEmojisUpdateEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		Emojis  []discord.Emoji   `json:"emoji"`
	}

	GuildIntegrationsUpdateEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
	}

	GuildMemberAddEvent struct {
		discord.Member
		GuildID discord.Snowflake `json:"guild_id"`
	}
	GuildMemberRemoveEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		User    discord.User      `json:"user"`
	}
	GuildMemberUpdateEvent struct {
		GuildID discord.Snowflake   `json:"guild_id"`
		RoleIDs []discord.Snowflake `json:"roles"`
		User    discord.User        `json:"user"`
		Nick    string              `json:"nick"`
	}

	// GuildMembersChunkEvent is sent when Guild Request Members is called.
	GuildMembersChunkEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		Members []discord.Member  `json:"members"`

		// Whatever's not found goes here
		NotFound []string `json:"not_found,omitempty"`

		// Only filled if requested
		Presences []discord.Presence `json:"presences,omitempty"`
	}

	// GuildMemberListUpdate is an undocumented event. It's received when the
	// client sends over GuildSubscriptions with the Channels field used.
	// The State package does not handle this event.
	GuildMemberListUpdate struct {
		ID          string            `json:"id"`
		GuildID     discord.Snowflake `json:"guild_id"`
		MemberCount uint64            `json:"member_count"`
		OnlineCount uint64            `json:"online_count"`

		// Groups is all the visible role sections.
		Groups []GuildMemberListGroup `json:"groups"`

		Ops []GuildMemberListOp `json:"ops"`
	}
	GuildMemberListGroup struct {
		ID    string `json:"id"` // either discord.Snowflake Role IDs or "online"
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
		GuildID discord.Snowflake `json:"guild_id"`
		Role    discord.Role      `json:"role"`
	}
	GuildRoleUpdateEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		Role    discord.Role      `json:"role"`
	}
	GuildRoleDeleteEvent struct {
		GuildID discord.Snowflake `json:"guild_id"`
		RoleID  discord.Snowflake `json:"role_id"`
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
		ChannelID discord.Snowflake `json:"channel_id"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`

		// Similar to discord.Invite
		Inviter    *discord.User          `json:"inviter,omitempty"`
		Target     *discord.User          `json:"target_user,omitempty"`
		TargetType discord.InviteUserType `json:"target_user_type,omitempty"`

		discord.InviteMetadata
	}
	InviteDeleteEvent struct {
		Code      string            `json:"code"`
		ChannelID discord.Snowflake `json:"channel_id"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
	}
)

// https://discordapp.com/developers/docs/topics/gateway#messages
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
		ID        discord.Snowflake `json:"id"`
		ChannelID discord.Snowflake `json:"channel_id"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
	}
	MessageDeleteBulkEvent struct {
		IDs       []discord.Snowflake `json:"ids"`
		ChannelID discord.Snowflake   `json:"channel_id"`
		GuildID   discord.Snowflake   `json:"guild_id,omitempty"`
	}

	MessageReactionAddEvent struct {
		UserID    discord.Snowflake `json:"user_id"`
		ChannelID discord.Snowflake `json:"channel_id"`
		MessageID discord.Snowflake `json:"message_id"`

		Emoji discord.Emoji `json:"emoji,omitempty"`

		GuildID discord.Snowflake `json:"guild_id,omitempty"`
		Member  *discord.Member   `json:"member,omitempty"`
	}
	MessageReactionRemoveEvent struct {
		UserID    discord.Snowflake `json:"user_id"`
		ChannelID discord.Snowflake `json:"channel_id"`
		MessageID discord.Snowflake `json:"message_id"`
		Emoji     discord.Emoji     `json:"emoji"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveAllEvent struct {
		ChannelID discord.Snowflake `json:"channel_id"`
		MessageID discord.Snowflake `json:"message_id"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveEmoji struct {
		ChannelID discord.Snowflake `json:"channel_id"`
		MessageID discord.Snowflake `json:"message_id"`
		Emoji     discord.Emoji     `json:"emoji"`
		GuildID   discord.Snowflake `json:"guild_id,omitempty"`
	}

	MessageAckEvent struct {
		MessageID discord.Snowflake `json:"message_id"`
		ChannelID discord.Snowflake `json:"channel_id"`
	}
)

// https://discordapp.com/developers/docs/topics/gateway#presence
type (
	// Clients may only update their game status 5 times per 20 seconds.
	PresenceUpdateEvent struct {
		discord.Presence
	}
	PresencesReplaceEvent []discord.Presence

	// SessionsReplaceEvent is an undocumented user event. It's likely used for
	// current user's presence updates.
	SessionsReplaceEvent []struct {
		Status    discord.Status `json:"status"`
		SessionID string         `json:"session_id"`

		Game       *discord.Activity  `json:"game"`
		Activities []discord.Activity `json:"activities"`

		ClientInfo struct {
			Version int    `json:"version"`
			OS      string `json:"os"`
			Client  string `json:"client"`
		} `json:"client_info"`

		Active bool `json:"active"`
	}

	TypingStartEvent struct {
		ChannelID discord.Snowflake     `json:"channel_id"`
		UserID    discord.Snowflake     `json:"user_id"`
		Timestamp discord.UnixTimestamp `json:"timestamp"`

		GuildID discord.Snowflake `json:"guild_id,omitempty"`
		Member  *discord.Member   `json:"member,omitempty"`
	}

	UserUpdateEvent struct {
		discord.User
	}
)

// https://discordapp.com/developers/docs/topics/gateway#voice
type (
	VoiceStateUpdateEvent struct {
		discord.VoiceState
	}
	VoiceServerUpdateEvent struct {
		Token    string            `json:"token"`
		GuildID  discord.Snowflake `json:"guild_id"`
		Endpoint string            `json:"endpoint"`
	}
)

// https://discordapp.com/developers/docs/topics/gateway#webhooks
type (
	WebhooksUpdateEvent struct {
		GuildID   discord.Snowflake `json:"guild_id"`
		ChannelID discord.Snowflake `json:"channel_id"`
	}
)

// Undocumented
type (
	UserGuildSettingsUpdateEvent struct {
		UserGuildSettings
	}
	UserSettingsUpdateEvent struct {
		UserSettings
	}
	UserNoteUpdateEvent struct {
		ID   discord.Snowflake `json:"id"`
		Note string            `json:"note"`
	}
)

type (
	RelationshipAdd struct {
		Relationship
	}
	RelationshipRemove struct {
		Relationship
	}
)
