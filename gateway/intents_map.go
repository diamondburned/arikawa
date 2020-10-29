package gateway

// EventIntents maps event types to intents.
var EventIntents = map[string]Intents{
	"GUILD_CREATE":        IntentGuilds,
	"GUILD_UPDATE":        IntentGuilds,
	"GUILD_DELETE":        IntentGuilds,
	"GUILD_ROLE_CREATE":   IntentGuilds,
	"GUILD_ROLE_UPDATE":   IntentGuilds,
	"GUILD_ROLE_DELETE":   IntentGuilds,
	"CHANNEL_CREATE":      IntentGuilds,
	"CHANNEL_UPDATE":      IntentGuilds,
	"CHANNEL_DELETE":      IntentGuilds,
	"CHANNEL_PINS_UPDATE": IntentGuilds | IntentDirectMessages,

	"GUILD_MEMBER_ADD":    IntentGuildMembers,
	"GUILD_MEMBER_REMOVE": IntentGuildMembers,
	"GUILD_MEMBER_UPDATE": IntentGuildMembers,

	"GUILD_BAN_ADD":    IntentGuildBans,
	"GUILD_BAN_REMOVE": IntentGuildBans,

	"GUILD_EMOJIS_UPDATE": IntentGuildEmojis,

	"GUILD_INTEGRATIONS_UPDATE": IntentGuildIntegrations,

	"WEBHOOKS_UPDATE": IntentGuildWebhooks,

	"INVITE_CREATE": IntentGuildInvites,
	"INVITE_DELETE": IntentGuildInvites,

	"VOICE_STATE_UPDATE": IntentGuildVoiceStates,

	"PRESENCE_UPDATE": IntentGuildPresences,

	"MESSAGE_CREATE":      IntentGuildMessages | IntentDirectMessages,
	"MESSAGE_UPDATE":      IntentGuildMessages | IntentDirectMessages,
	"MESSAGE_DELETE":      IntentGuildMessages | IntentDirectMessages,
	"MESSAGE_DELETE_BULK": IntentGuildMessages,

	"MESSAGE_REACTION_ADD":          IntentGuildMessageReactions | IntentDirectMessageReactions,
	"MESSAGE_REACTION_REMOVE":       IntentGuildMessageReactions | IntentDirectMessageReactions,
	"MESSAGE_REACTION_REMOVE_ALL":   IntentGuildMessageReactions | IntentDirectMessageReactions,
	"MESSAGE_REACTION_REMOVE_EMOJI": IntentGuildMessageReactions | IntentDirectMessageReactions,

	"TYPING_START": IntentGuildMessageTyping | IntentDirectMessageTyping,
}
