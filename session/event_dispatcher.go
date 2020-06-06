package session

import "github.com/diamondburned/arikawa/gateway"

func (s *Session) handleEvent(ev interface{}) {
	switch e := ev.(type) {
	case *gateway.ReadyEvent:
		s.handleReady(e)
	case *gateway.GuildCreateEvent:
		s.handleGuildCreate(e)
	case *gateway.GuildDeleteEvent:
		s.handleGuildDelete(e)
	default:
		s.Handler.Call(e)
	}
}

func (s *Session) handleReady(ev *gateway.ReadyEvent) {
	s.Handler.Call(ev)

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

func (s *Session) handleGuildCreate(ev *gateway.GuildCreateEvent) {
	// before we dispatch the specific events, we can already call the handlers
	// that subscribed to the generic version
	s.Handler.CallDirect(ev)

	// this guild was unavailable, but has come back online
	if s.unavailableGuilds.Delete(ev.ID) {
		s.Handler.Call(&GuildAvailableEvent{
			GuildCreateEvent: ev,
		})

		// the guild was already unavailable when connecting to the gateway
		// we can dispatch a belated GuildReadyEvent
	} else if s.unreadyGuilds.Delete(ev.ID) {
		s.Handler.Call(&GuildReadyEvent{
			GuildCreateEvent: ev,
		})
	} else { // we don't know this guild, hence we just joined it
		s.Handler.Call(&GuildJoinEvent{
			GuildCreateEvent: ev,
		})
	}
}

func (s *Session) handleGuildDelete(ev *gateway.GuildDeleteEvent) {
	// before we dispatch the specific events, we can already call the handlers
	// that subscribed to the generic version
	s.Handler.CallDirect(ev)

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
