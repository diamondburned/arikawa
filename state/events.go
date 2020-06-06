package state

import "github.com/diamondburned/arikawa/gateway"

// events that originated from GuildCreate:
type (
	// GuildReady gets fired for every guild the bot/user is in, as found in
	// the Ready event.
	//
	// Guilds that are unavailable when connecting, will not trigger a
	// GuildReadyEvent, until they become available again.
	GuildReadyEvent struct {
		*gateway.GuildCreateEvent
	}

	// GuildAvailableEvent gets fired when a guild becomes available again,
	// after being previously declared unavailable through a
	// GuildUnavailableEvent. This event will not be fired for guilds that
	// were already unavailable when connecting to the gateway.
	GuildAvailableEvent struct {
		*gateway.GuildCreateEvent
	}

	// GuildJoinEvent gets fired if the bot/user joins a guild.
	GuildJoinEvent struct {
		*gateway.GuildCreateEvent
	}
)

// events that originated from GuildDelete:
type (
	// GuildLeaveEvent gets fired if the bot/user left a guild, was removed
	// or the owner deleted the guild.
	GuildLeaveEvent struct {
		*gateway.GuildDeleteEvent
	}

	// GuildUnavailableEvent gets fired if a guild becomes unavailable.
	GuildUnavailableEvent struct {
		*gateway.GuildDeleteEvent
	}
)
