package gateway

// Intents for the new Discord API feature, documented at
// https://discordapp.com/developers/docs/topics/gateway#gateway-intents.
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
