// Package voice handles the Discord voice gateway and UDP connections. It does
// not handle book-keeping of those sessions.
//
// This package abstracts the subpackage voice/voicesession and voice/udp.
package voice

import "github.com/diamondburned/arikawa/v3/gateway"

// Intents are the gateway.Intents need to operate a Session. Bots should
// always add these before opening.
const Intents = gateway.IntentGuilds | gateway.IntentGuildVoiceStates
