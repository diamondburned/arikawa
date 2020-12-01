// Package voice handles the Discord voice gateway and UDP connections. It does
// not handle book-keeping of those sessions.
//
// This package abstracts the subpackage voice/voicesession and voice/udp.
package voice

import "github.com/diamondburned/arikawa/v2/gateway"

// AddIntents adds the needed voice intents into gw. Bots should always call
// this before Open if voice is required.
func AddIntents(gw *gateway.Gateway) {
	gw.AddIntents(gateway.IntentGuilds)
	gw.AddIntents(gateway.IntentGuildVoiceStates)
}
