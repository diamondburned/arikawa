package state

import (
	"libdb.so/arikawa/v4/gateway"
)

func (s *State) handleReady(ev *gateway.ReadyEvent) {
	s.guildMutex.Lock()
	defer s.guildMutex.Unlock()

	for chID := range s.fewMessages {
		delete(s.fewMessages, chID)
	}

	for _, g := range ev.Guilds {
		s.unreadyGuilds[g.ID] = struct{}{}
	}
}

func (s *State) handleGuildCreate(ev *gateway.GuildCreateEvent) {
	s.guildMutex.Lock()

	var derivedEvent interface{}

	// The guild was previously announced to us in the ready event, and has now
	// become available.
	if _, ok := s.unreadyGuilds[ev.ID]; ok {
		delete(s.unreadyGuilds, ev.ID)
		derivedEvent = &GuildReadyEvent{GuildCreateEvent: ev}

		// The guild was previously announced as unavailable through a guild
		// delete event, and has now become available again.
	} else if _, ok = s.unavailableGuilds[ev.ID]; ok {
		delete(s.unavailableGuilds, ev.ID)
		derivedEvent = &GuildAvailableEvent{GuildCreateEvent: ev}

		// We don't know this guild, hence it's new.
	} else {
		derivedEvent = &GuildJoinEvent{GuildCreateEvent: ev}
	}

	// Unlock here already, so we don't block the mutex if there are
	// long-blocking synchronous handlers.
	s.guildMutex.Unlock()
	s.Handler.Call(derivedEvent)
}

func (s *State) handleGuildDelete(ev *gateway.GuildDeleteEvent) {
	s.guildMutex.Lock()

	// store this so we can later dispatch a GuildAvailableEvent, once the
	// guild becomes available again.
	if ev.Unavailable {
		s.unavailableGuilds[ev.ID] = struct{}{}
		s.guildMutex.Unlock()

		s.Handler.Call(&GuildUnavailableEvent{GuildDeleteEvent: ev})
	} else {
		// Possible scenario requiring this would be leaving the guild while
		// unavailable.
		delete(s.unavailableGuilds, ev.ID)
		s.guildMutex.Unlock()

		s.Handler.Call(&GuildLeaveEvent{GuildDeleteEvent: ev})
	}
}
