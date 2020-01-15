package gateway

// Event is any event struct. They have an "Event" suffixed to them.
type Event = interface{}

var EventCreator = map[string]func() Event{
	"HELLO":           func() Event { return new(HelloEvent) },
	"READY":           func() Event { return new(ReadyEvent) },
	"RESUMED":         func() Event { return new(ResumedEvent) },
	"INVALID_SESSION": func() Event { return new(InvalidSessionEvent) },

	"CHANNEL_CREATE":      func() Event { return new(ChannelCreateEvent) },
	"CHANNEL_UPDATE":      func() Event { return new(ChannelUpdateEvent) },
	"CHANNEL_DELETE":      func() Event { return new(ChannelDeleteEvent) },
	"CHANNEL_PINS_UPDATE": func() Event { return new(ChannelPinsUpdateEvent) },

	"GUILD_CREATE": func() Event { return new(GuildCreateEvent) },
	"GUILD_UPDATE": func() Event { return new(GuildUpdateEvent) },
	"GUILD_DELETE": func() Event { return new(GuildDeleteEvent) },

	"GUILD_BAN_ADD":    func() Event { return new(GuildBanAddEvent) },
	"GUILD_BAN_REMOVE": func() Event { return new(GuildBanRemoveEvent) },

	"GUILD_EMOJIS_UPDATE": func() Event { return new(GuildEmojisUpdateEvent) },
	"GUILD_INTEGRATIONS_UPDATE": func() Event {
		return new(GuildIntegrationsUpdateEvent)
	},

	"GUILD_MEMBER_ADD":    func() Event { return new(GuildMemberAddEvent) },
	"GUILD_MEMBER_REMOVE": func() Event { return new(GuildMemberRemoveEvent) },
	"GUILD_MEMBER_UPDATE": func() Event { return new(GuildMemberUpdateEvent) },
	"GUILD_MEMBERS_CHUNK": func() Event { return new(GuildMembersChunkEvent) },

	"GUILD_ROLE_CREATE": func() Event { return new(GuildRoleCreateEvent) },
	"GUILD_ROLE_UPDATE": func() Event { return new(GuildRoleUpdateEvent) },
	"GUILD_ROLE_DELETE": func() Event { return new(GuildRoleDeleteEvent) },

	"MESSAGE_CREATE":      func() Event { return new(MessageCreateEvent) },
	"MESSAGE_UPDATE":      func() Event { return new(MessageUpdateEvent) },
	"MESSAGE_DELETE":      func() Event { return new(MessageDeleteEvent) },
	"MESSAGE_DELETE_BULK": func() Event { return new(MessageDeleteBulkEvent) },

	"MESSAGE_REACTION_ADD": func() Event {
		return new(MessageReactionAddEvent)
	},
	"MESSAGE_REACTION_REMOVE": func() Event {
		return new(MessageReactionRemoveEvent)
	},
	"MESSAGE_REACTION_REMOVE_ALL": func() Event {
		return new(MessageReactionRemoveAllEvent)
	},

	"PRESENCE_UPDATE": func() Event { return new(PresenceUpdateEvent) },
	"TYPING_START":    func() Event { return new(TypingStartEvent) },
	"USER_UPDATE":     func() Event { return new(UserUpdateEvent) },

	"VOICE_STATE_UPDATE":  func() Event { return new(VoiceStateUpdateEvent) },
	"VOICE_SERVER_UPDATE": func() Event { return new(VoiceServerUpdateEvent) },
}
