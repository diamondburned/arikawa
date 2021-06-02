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
