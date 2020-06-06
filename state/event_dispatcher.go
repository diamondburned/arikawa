package state

import (
	"github.com/diamondburned/arikawa/gateway"
)

func (s *State) handleEvent(ev interface{}) {
	if s.PreHandler != nil {
		s.PreHandler.Call(ev)
	}
	s.Handler.Call(ev)
}

func (s *State) handleReady(ev *gateway.ReadyEvent) {
	s.handleEvent(ev)

	for _, g := range ev.Guilds {
		// store this so we know when we need to dispatch a belated
		// GuildReadyEvent
		if g.Unavailable {
			s.unreadyGuilds.Add(g.ID)
		} else {
			s.handleEvent(&GuildReadyEvent{
				GuildCreateEvent: &g,
			})
		}
	}
}

func (s *State) handleGuildCreate(ev *gateway.GuildCreateEvent) {
	// before we dispatch the specific events, we can already call the handlers
	// that subscribed to the generic version
	s.handleEvent(ev)

	// this guild was unavailable, but has come back online
	if s.unavailableGuilds.Delete(ev.ID) {
		s.handleEvent(&GuildAvailableEvent{
			GuildCreateEvent: ev,
		})

		// the guild was already unavailable when connecting to the gateway
		// we can dispatch a belated GuildReadyEvent
	} else if s.unreadyGuilds.Delete(ev.ID) {
		s.handleEvent(&GuildReadyEvent{
			GuildCreateEvent: ev,
		})
	} else { // we don't know this guild, hence we just joined it
		s.handleEvent(&GuildJoinEvent{
			GuildCreateEvent: ev,
		})
	}
}

func (s *State) handleGuildDelete(ev *gateway.GuildDeleteEvent) {
	// before we dispatch the specific events, we can already call the handlers
	// that subscribed to the generic version
	s.handleEvent(ev)

	// store this so we can later dispatch a GuildAvailableEvent, once the
	// guild becomes available again.
	if ev.Unavailable {
		s.unavailableGuilds.Add(ev.ID)

		s.handleEvent(&GuildUnavailableEvent{
			GuildDeleteEvent: ev,
		})
	} else {
		// it might have been unavailable before we left
		s.unavailableGuilds.Delete(ev.ID)

		s.handleEvent(&GuildLeaveEvent{
			GuildDeleteEvent: ev,
		})
	}
}
