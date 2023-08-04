// Package voice handles the Discord voice gateway and UDP connections. It does
// not handle book-keeping of those sessions.
//
// This package abstracts the subpackage voice/voicesession and voice/udp.
package voice

import "libdb.so/arikawa/v4/gateway"

// Intents are the intents needed for voice to work properly.
const Intents = gateway.IntentGuilds | gateway.IntentGuildVoiceStates

// AddIntents adds the needed voice intents into gw. Bots should always call
// this before Open if voice is required.
func AddIntents(gw interface{ AddIntents(gateway.Intents) }) {
	gw.AddIntents(Intents)
}
