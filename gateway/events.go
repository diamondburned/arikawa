package gateway

import "github.com/diamondburned/arikawa/v2/discord"

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
		LastPin   discord.Timestamp `json:"timestamp,omitempty"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
		ChannelID discord.ChannelID `json:"channel_id,omitempty"`
	}

	ChannelUnreadUpdateEvent struct {
		ChannelUnreadUpdates []struct {
			ID            discord.ChannelID `json:"id"`
			LastMessageID discord.MessageID `json:"last_message_id"`
		}
		GuildID discord.GuildID `json:"guild_id"`
	}
)

// https://discord.com/developers/docs/topics/gateway#guilds
type (
	GuildCreateEvent struct {
		Joined      discord.Timestamp    `json:"joined_at,omitempty"`
		Members     []discord.Member     `json:"members,omitempty"`
		Presences   []Presence           `json:"presences,omitempty"`
		VoiceStates []discord.VoiceState `json:"voice_states,omitempty"`
		Channels    []discord.Channel    `json:"channels,omitempty"`
		discord.Guild
		MemberCount uint64 `json:"member_count,omitempty"`
		Unavailable bool   `json:"unavailable,omitempty"`
		Large       bool   `json:"large,omitempty"`
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
		User    discord.User    `json:"user"`
		GuildID discord.GuildID `json:"guild_id"`
	}
	GuildBanRemoveEvent struct {
		User    discord.User    `json:"user"`
		GuildID discord.GuildID `json:"guild_id"`
	}

	GuildEmojisUpdateEvent struct {
		Emojis  []discord.Emoji `json:"emoji"`
		GuildID discord.GuildID `json:"guild_id"`
	}

	GuildIntegrationsUpdateEvent struct {
		GuildID discord.GuildID `json:"guild_id"`
	}

	GuildMemberAddEvent struct {
		discord.Member
		GuildID discord.GuildID `json:"guild_id"`
	}
	GuildMemberRemoveEvent struct {
		User    discord.User    `json:"user"`
		GuildID discord.GuildID `json:"guild_id"`
	}
	GuildMemberUpdateEvent struct {
		Nick    string           `json:"nick"`
		RoleIDs []discord.RoleID `json:"roles"`
		User    discord.User     `json:"user"`
		GuildID discord.GuildID  `json:"guild_id"`
	}

	// GuildMembersChunkEvent is sent when Guild Request Members is called.
	GuildMembersChunkEvent struct {
		Nonce   string           `json:"nonce,omitempty"`
		Members []discord.Member `json:"members"`
		// Whatever's not found goes here
		NotFound []string `json:"not_found,omitempty"`
		// Only filled if requested
		Presences  []Presence      `json:"presences,omitempty"`
		ChunkIndex int             `json:"chunk_index"`
		ChunkCount int             `json:"chunk_count"`
		GuildID    discord.GuildID `json:"guild_id"`
	}

	// GuildMemberListUpdate is an undocumented event. It's received when the
	// client sends over GuildSubscriptions with the Channels field used.
	// The State package does not handle this event.
	GuildMemberListUpdate struct {
		ID string `json:"id"`
		// Groups is all the visible role sections.
		Groups      []GuildMemberListGroup `json:"groups"`
		Ops         []GuildMemberListOp    `json:"ops"`
		GuildID     discord.GuildID        `json:"guild_id"`
		MemberCount uint64                 `json:"member_count"`
		OnlineCount uint64                 `json:"online_count"`
	}
	GuildMemberListGroup struct {
		ID    string `json:"id"` // either discord.RoleID, "online" or "offline"
		Count uint64 `json:"count"`
	}
	GuildMemberListOp struct {
		Item GuildMemberListOpItem `json:"item,omitempty"`
		// Mysterious string, so far spotted to be [SYNC, INSERT, UPDATE, DELETE].
		Op string `json:"op"`
		// Items is basically a linear list of roles and members, similarly to
		// how the client renders it. No, it's not nested.
		Items []GuildMemberListOpItem `json:"items,omitempty"`
		// SYNC ONLY
		// Range requested in GuildSubscribeData.
		Range [2]int `json:"range,omitempty"`
		// NON-SYNC ONLY
		// Only available for Ops that aren't "SYNC".
		Index int `json:"index,omitempty"`
	}
	// GuildMemberListOpItem is an enum. Either of the fields are provided, but
	// never both. Refer to (*GuildMemberListUpdate).Ops for more.
	GuildMemberListOpItem struct {
		Group  *GuildMemberListGroup `json:"group,omitempty"`
		Member *struct {
			discord.Member
			HoistedRole string   `json:"hoisted_role"`
			Presence    Presence `json:"presence"`
		} `json:"member,omitempty"`
	}

	GuildRoleCreateEvent struct {
		Role    discord.Role    `json:"role"`
		GuildID discord.GuildID `json:"guild_id"`
	}
	GuildRoleUpdateEvent struct {
		Role    discord.Role    `json:"role"`
		GuildID discord.GuildID `json:"guild_id"`
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
		CreatedAt discord.Timestamp `json:"created_at"`
		// Similar to discord.Invite
		Inviter *discord.User `json:"inviter,omitempty"`
		Target  *discord.User `json:"target_user,omitempty"`
		Code    string        `json:"code"`
		discord.InviteMetadata
		ChannelID  discord.ChannelID      `json:"channel_id"`
		GuildID    discord.GuildID        `json:"guild_id,omitempty"`
		TargetType discord.InviteUserType `json:"target_user_type,omitempty"`
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
		Member *discord.Member `json:"member,omitempty"`
		discord.Message
	}
	MessageUpdateEvent struct {
		Member *discord.Member `json:"member,omitempty"`
		discord.Message
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
		Member    *discord.Member   `json:"member,omitempty"`
		Emoji     discord.Emoji     `json:"emoji,omitempty"`
		UserID    discord.UserID    `json:"user_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveEvent struct {
		Emoji     discord.Emoji     `json:"emoji"`
		UserID    discord.UserID    `json:"user_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveAllEvent struct {
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}
	MessageReactionRemoveEmojiEvent struct {
		Emoji     discord.Emoji     `json:"emoji"`
		ChannelID discord.ChannelID `json:"channel_id"`
		MessageID discord.MessageID `json:"message_id"`
		GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	}

	MessageAckEvent struct {
		MessageID discord.MessageID `json:"message_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
	}
)

// Status is the enumerate type for a user's status.
type Status string

const (
	UnknownStatus      Status = ""
	OnlineStatus       Status = "online"
	DoNotDisturbStatus Status = "dnd"
	IdleStatus         Status = "idle"
	InvisibleStatus    Status = "invisible"
	OfflineStatus      Status = "offline"
)

// https://discord.com/developers/docs/topics/gateway#presence
type (
	// Presence represents a partial Presence structure used by other structs to be
	// easily embedded. It does not contain any ID to identify who it belongs
	// to. For more information, refer to the PresenceUpdateEvent struct.
	Presence struct {
		// ClientStatus is the user's platform-dependent status.
		ClientStatus ClientStatus `json:"client_status"`
		// Status is either "idle", "dnd", "online", or "offline".
		Status Status `json:"status"`
		// Activities are the user's current activities.
		Activities []discord.Activity `json:"activities"`
		// User is the user presence is being updated for. Only the ID field is
		// guaranteed to be valid per Discord documentation.
		User discord.User `json:"user"`
		// GuildID is the id of the guild
		GuildID discord.GuildID `json:"guild_id"`
	}

	// ClientStatus is the user's platform-dependent status.
	//
	// https://discord.com/developers/docs/topics/gateway#client-status-object
	ClientStatus struct {
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

	// PresenceUpdateEvent represents the structure of the Presence Update Event
	// object.
	//
	// https://discord.com/developers/docs/topics/gateway#presence-update-presence-update-event-fields
	PresenceUpdateEvent struct {
		Presence
	}

	PresencesReplaceEvent []PresenceUpdateEvent

	// SessionsReplaceEvent is an undocumented user event. It's likely used for
	// current user's presence updates.
	SessionsReplaceEvent []struct {
		ClientInfo struct {
			OS      string `json:"os"`
			Client  string `json:"client"`
			Version int    `json:"version"`
		} `json:"client_info"`
		Status     Status             `json:"status"`
		SessionID  string             `json:"session_id"`
		Activities []discord.Activity `json:"activities"`
		Active     bool               `json:"active"`
	}

	TypingStartEvent struct {
		Member    *discord.Member       `json:"member,omitempty"`
		ChannelID discord.ChannelID     `json:"channel_id"`
		UserID    discord.UserID        `json:"user_id"`
		Timestamp discord.UnixTimestamp `json:"timestamp"`
		GuildID   discord.GuildID       `json:"guild_id,omitempty"`
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
		Endpoint string          `json:"endpoint"`
		GuildID  discord.GuildID `json:"guild_id"`
	}
)

// https://discord.com/developers/docs/topics/gateway#webhooks
type (
	WebhooksUpdateEvent struct {
		GuildID   discord.GuildID   `json:"guild_id"`
		ChannelID discord.ChannelID `json:"channel_id"`
	}
)

// https://discord.com/developers/docs/topics/gateway#interactions
type (
	InteractionCreateEvent struct {
		Message *discord.Message `json:"message"`
		// Member is only present if this came from a guild.
		Member *discord.Member  `json:"member,omitempty"`
		Data   *InteractionData `json:"data,omitempty"`
		// User is only present if this didn't come from a guild.
		User      *discord.User         `json:"user,omitempty"`
		Token     string                `json:"token"`
		ID        discord.InteractionID `json:"id"`
		Version   int                   `json:"version"`
		AppID     discord.AppID         `json:"application_id"`
		Type      InteractionType       `json:"type"`
		GuildID   discord.GuildID       `json:"guild_id,omitempty"`
		ChannelID discord.ChannelID     `json:"channel_id,omitempty"`
	}
)

type InteractionType uint

const (
	PingInteraction InteractionType = iota + 1
	CommandInteraction
	ButtonInteraction
)

// TODO: InteractionData is being overloaded by Slash Command and Button at the moment.
//       Separate them when v3 rolls out.

type InteractionData struct {
	// Slash commands
	Name string `json:"name"`
	// Button
	CustomID string `json:"custom_id"`
	// Slash commands
	Options []InteractionOption `json:"options"`
	// Slash commands
	ID discord.CommandID `json:"id"`
	// Button
	ComponentType discord.ComponentType `json:"component_type"`
}

type InteractionOption struct {
	Name    string              `json:"name"`
	Value   string              `json:"value"`
	Options []InteractionOption `json:"options"`
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
		Note string         `json:"note"`
		ID   discord.UserID `json:"id"`
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
