package voice

import (
	"github.com/diamondburned/arikawa/gateway"
)

// onVoiceStateUpdate receives VoiceStateUpdateEvents from the gateway
// to keep track of the current user's voice state.
func (v *Voice) onVoiceStateUpdate(e *gateway.VoiceStateUpdateEvent) {
	// Get the current user.
	me, err := v.state.Me()
	if err != nil {
		v.ErrorLog(err)
		return
	}

	// Ignore the event if it is an update from another user.
	if me.ID != e.UserID {
		return
	}

	// Get the stored voice connection for the given guild.
	conn, ok := v.GetConnection(e.GuildID)

	// Ignore if there is no connection for that guild.
	if !ok {
		return
	}

	// Remove the connection if the current user has disconnected.
	if e.ChannelID == 0 {
		// TODO: Make sure connection is closed?
		v.RemoveConnection(e.GuildID)
		return
	}

	// Update values on the connection.
	conn.mut.Lock()
	defer conn.mut.Unlock()

	conn.SessionID = e.SessionID

	conn.UserID = e.UserID
	conn.ChannelID = e.ChannelID
}

// onVoiceServerUpdate receives VoiceServerUpdateEvents from the gateway
// to manage the current user's voice connections.
func (v *Voice) onVoiceServerUpdate(e *gateway.VoiceServerUpdateEvent) {
	// Get the stored voice connection for the given guild.
	conn, ok := v.GetConnection(e.GuildID)

	// Ignore if there is no connection for that guild.
	if !ok {
		return
	}

	// Ensure the connection is closed (has no effect if the connection is already closed)
	conn.Close()

	// Update values on the connection.
	conn.mut.Lock()
	conn.Token = e.Token
	conn.Endpoint = e.Endpoint

	conn.GuildID = e.GuildID
	conn.mut.Unlock()

	// Open the voice connection.
	if err := conn.Open(); err != nil {
		v.ErrorLog(err)
		return
	}
}
