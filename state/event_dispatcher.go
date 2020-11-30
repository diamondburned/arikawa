package state

import (
	"github.com/diamondburned/arikawa/v2/gateway"
)

func (s *State) handleReady(ev *gateway.ReadyEvent) {
	for _, g := range ev.Guilds {
		// store this so we know when we need to dispatch a belated
		// GuildReadyEvent
		if g.Unavailable {
			s.unreadyGuilds.Add(g.ID)
		} else {
			s.Handler.Call(&GuildReadyEvent{
				GuildCreateEvent: &g,
			})
		}
	}
}

func (s *State) handleGuildCreate(ev *gateway.GuildCreateEvent) {
	switch {
	// this guild was unavailable, but has come back online
	case s.unavailableGuilds.Delete(ev.ID):
		s.Handler.Call(&GuildAvailableEvent{
			GuildCreateEvent: ev,
		})

	// the guild was already unavailable when connecting to the gateway
	// we can dispatch a belated GuildReadyEvent
	case s.unreadyGuilds.Delete(ev.ID):
		s.Handler.Call(&GuildReadyEvent{
			GuildCreateEvent: ev,
		})

	// we don't know this guild, hence we just joined it
	default:
		s.Handler.Call(&GuildJoinEvent{
			GuildCreateEvent: ev,
		})
	}
}

func (s *State) handleGuildDelete(ev *gateway.GuildDeleteEvent) {
	// store this so we can later dispatch a GuildAvailableEvent, once the
	// guild becomes available again.
	if ev.Unavailable {
		s.unavailableGuilds.Add(ev.ID)

		s.Handler.Call(&GuildUnavailableEvent{
			GuildDeleteEvent: ev,
		})
	} else {
		// it might have been unavailable before we left
		s.unavailableGuilds.Delete(ev.ID)

		s.Handler.Call(&GuildLeaveEvent{
			GuildDeleteEvent: ev,
		})
	}
}
