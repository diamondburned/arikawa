package gateway

import "github.com/diamondburned/arikawa/v3/discord"

// Intents for the new Discord API feature, documented at
// https://discord.com/developers/docs/topics/gateway#gateway-intents.
type Intents uint32

const (
	IntentGuilds Intents = 1 << iota
	IntentGuildMembers
	IntentGuildBans
	IntentGuildEmojis
	IntentGuildIntegrations
	IntentGuildWebhooks
	IntentGuildInvites
	IntentGuildVoiceStates
	IntentGuildPresences
	IntentGuildMessages
	IntentGuildMessageReactions
	IntentGuildMessageTyping
	IntentDirectMessages
	IntentDirectMessageReactions
	IntentDirectMessageTyping
)

// PrivilegedIntents contains a list of privileged intents that Discord requires
// bots to have these intents explicitly enabled in the Developer Portal.
var PrivilegedIntents = []Intents{
	IntentGuildPresences,
	IntentGuildMembers,
}

// Has returns true if i has the given intents.
func (i Intents) Has(intents Intents) bool {
	return discord.HasFlag(uint64(i), uint64(intents))
}

// IsPrivileged returns true for each of the boolean that indicates the type of
// the privilege.
func (i Intents) IsPrivileged() (presences, member bool) {
	// Keep this in sync with PrivilegedIntents.
	return i.Has(IntentGuildPresences), i.Has(IntentGuildMembers)
}

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
