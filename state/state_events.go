package state

import (
	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state/store"
)

func (s *State) hookSession() {
	s.Session.AddHandler(func(event interface{}) {
		// Call the pre-handler before the state handler.
		if s.PreHandler != nil {
			s.PreHandler.Call(event)
		}

		// Run the state handler.
		s.onEvent(event)

		switch event := event.(type) {
		case *gateway.ReadyEvent:
			s.Handler.Call(event)
			s.handleReady(event)
		case *gateway.GuildCreateEvent:
			s.Handler.Call(event)
			s.handleGuildCreate(event)
		case *gateway.GuildDeleteEvent:
			s.Handler.Call(event)
			s.handleGuildDelete(event)

		// https://github.com/discord/discord-api-docs/commit/01665c4
		case *gateway.MessageCreateEvent:
			if event.Member != nil {
				event.Member.User = event.Author
			}
			s.Handler.Call(event)

		case *gateway.MessageUpdateEvent:
			if event.Member != nil {
				event.Member.User = event.Author
			}
			s.Handler.Call(event)

		default:
			s.Handler.Call(event)
		}
	})
}

func (s *State) onEvent(iface interface{}) {
	switch ev := iface.(type) {
	case *gateway.ReadyEvent:
		// Acquire the ready mutex, but since we're only writing the value and
		// not anything in it, we should be fine.
		s.readyMu.Lock()
		s.ready = *ev
		s.readyMu.Unlock()

		// Reset the store before proceeding.
		if err := s.Cabinet.Reset(); err != nil {
			s.stateErr(err, "failed to reset state in Ready")
		}

		// Handle guilds
		for i := range ev.Guilds {
			s.batchLog(storeGuildCreate(s.Cabinet, &ev.Guilds[i]))
		}

		// Handle guild presences
		for _, p := range ev.Presences {
			if err := s.Cabinet.PresenceSet(p.GuildID, p, false); err != nil {
				s.stateErr(err, "failed to set presence in Ready")
			}
		}

		// Handle private channels
		for _, ch := range ev.PrivateChannels {
			if err := s.Cabinet.ChannelSet(ch, false); err != nil {
				s.stateErr(err, "failed to set channel in Ready")
			}
		}

		// Handle user
		if err := s.Cabinet.MyselfSet(ev.User, false); err != nil {
			s.stateErr(err, "failed to set self in Ready")
		}

	case *gateway.ReadySupplementalEvent:
		// Handle guilds
		for _, guild := range ev.Guilds {
			// Handle guild voice states
			for _, v := range guild.VoiceStates {
				if err := s.Cabinet.VoiceStateSet(guild.ID, v, false); err != nil {
					s.stateErr(err, "failed to set guild voice state in Ready Supplemental")
				}
			}
		}

		for _, friend := range ev.MergedPresences.Friends {
			sPresence := gateway.ConvertSupplementalPresence(friend)
			if err := s.Cabinet.PresenceSet(0, sPresence, false); err != nil {
				s.stateErr(err, "failed to set friend presence in Ready Supplemental")
			}
		}

		// Discord uses weird indexing, so we'll need the Guilds slice.
		ready := s.Ready()

		for i := 0; i < len(ready.Guilds) && i < len(ev.MergedMembers); i++ {
			guild := ready.Guilds[i]

			for _, member := range ev.MergedMembers[i] {
				sMember := gateway.ConvertSupplementalMember(member)
				if err := s.Cabinet.MemberSet(guild.ID, sMember, false); err != nil {
					s.stateErr(err, "failed to set friend presence in Ready Supplemental")
				}
			}

			for _, member := range ev.MergedPresences.Guilds[i] {
				sPresence := gateway.ConvertSupplementalPresence(member)
				if err := s.Cabinet.PresenceSet(guild.ID, sPresence, false); err != nil {
					s.stateErr(err, "failed to set member presence in Ready Supplemental")
				}
			}
		}

	case *gateway.GuildCreateEvent:
		s.batchLog(storeGuildCreate(s.Cabinet, ev))

	case *gateway.GuildUpdateEvent:
		if err := s.Cabinet.GuildSet(ev.Guild, true); err != nil {
			s.stateErr(err, "failed to update guild in state")
		}

	case *gateway.GuildDeleteEvent:
		if err := s.Cabinet.GuildRemove(ev.ID); err != nil && !ev.Unavailable {
			s.stateErr(err, "failed to delete guild in state")
		}

	case *gateway.GuildMemberAddEvent:
		if err := s.Cabinet.MemberSet(ev.GuildID, ev.Member, false); err != nil {
			s.stateErr(err, "failed to add a member in state")
		}

	case *gateway.GuildMemberUpdateEvent:
		m, err := s.Cabinet.Member(ev.GuildID, ev.User.ID)
		if err != nil {
			// We can't do much here.
			m = &discord.Member{}
		}

		// Update available fields from ev into m
		ev.Update(m)

		if err := s.Cabinet.MemberSet(ev.GuildID, *m, true); err != nil {
			s.stateErr(err, "failed to update a member in state")
		}

	case *gateway.GuildMemberRemoveEvent:
		if err := s.Cabinet.MemberRemove(ev.GuildID, ev.User.ID); err != nil {
			s.stateErr(err, "failed to remove a member in state")
		}

	case *gateway.GuildMembersChunkEvent:
		for _, m := range ev.Members {
			if err := s.Cabinet.MemberSet(ev.GuildID, m, false); err != nil {
				s.stateErr(err, "failed to add a member from chunk in state")
			}
		}

		for _, p := range ev.Presences {
			if err := s.Cabinet.PresenceSet(ev.GuildID, p, false); err != nil {
				s.stateErr(err, "failed to add a presence from chunk in state")
			}
		}

	case *gateway.GuildRoleCreateEvent:
		if err := s.Cabinet.RoleSet(ev.GuildID, ev.Role, false); err != nil {
			s.stateErr(err, "failed to add a role in state")
		}

	case *gateway.GuildRoleUpdateEvent:
		if err := s.Cabinet.RoleSet(ev.GuildID, ev.Role, true); err != nil {
			s.stateErr(err, "failed to update a role in state")
		}

	case *gateway.GuildRoleDeleteEvent:
		if err := s.Cabinet.RoleRemove(ev.GuildID, ev.RoleID); err != nil {
			s.stateErr(err, "failed to remove a role in state")
		}

	case *gateway.GuildEmojisUpdateEvent:
		if err := s.Cabinet.EmojiSet(ev.GuildID, ev.Emojis, true); err != nil {
			s.stateErr(err, "failed to update emojis in state")
		}

	case *gateway.ChannelCreateEvent:
		if err := s.Cabinet.ChannelSet(ev.Channel, false); err != nil {
			s.stateErr(err, "failed to create a channel in state")
		}

	case *gateway.ChannelUpdateEvent:
		if err := s.Cabinet.ChannelSet(ev.Channel, true); err != nil {
			s.stateErr(err, "failed to update a channel in state")
		}

	case *gateway.ChannelDeleteEvent:
		if err := s.Cabinet.ChannelRemove(ev.Channel); err != nil {
			s.stateErr(err, "failed to remove a channel in state")
		}

	case *gateway.ChannelPinsUpdateEvent:
		// not tracked.

	case *gateway.MessageCreateEvent:
		if err := s.Cabinet.MessageSet(ev.Message, false); err != nil {
			s.stateErr(err, "failed to add a message in state")
		}

	case *gateway.MessageUpdateEvent:
		if err := s.Cabinet.MessageSet(ev.Message, true); err != nil {
			s.stateErr(err, "failed to update a message in state")
		}

	case *gateway.MessageDeleteEvent:
		if err := s.Cabinet.MessageRemove(ev.ChannelID, ev.ID); err != nil {
			s.stateErr(err, "failed to delete a message in state")
		}

	case *gateway.MessageDeleteBulkEvent:
		for _, id := range ev.IDs {
			if err := s.Cabinet.MessageRemove(ev.ChannelID, id); err != nil {
				s.stateErr(err, "failed to delete bulk messages in state")
			}
		}

	case *gateway.MessageReactionAddEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			if i := findReaction(m.Reactions, ev.Emoji); i > -1 {
				m.Reactions[i].Count++
			} else {
				var me bool
				if u, _ := s.Cabinet.Me(); u != nil {
					me = ev.UserID == u.ID
				}
				m.Reactions = append(m.Reactions, discord.Reaction{
					Count: 1,
					Me:    me,
					Emoji: ev.Emoji,
				})
			}
			return true
		})

	case *gateway.MessageReactionRemoveEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			var i = findReaction(m.Reactions, ev.Emoji)
			if i < 0 {
				return false
			}

			r := &m.Reactions[i]
			r.Count--

			switch {
			case r.Count < 1: // If the count is 0:
				// Remove the reaction.
				m.Reactions = append(m.Reactions[:i], m.Reactions[i+1:]...)

			case r.Me: // If reaction removal is the user's
				u, err := s.Cabinet.Me()
				if err == nil && ev.UserID == u.ID {
					r.Me = false
				}
			}

			return true
		})

	case *gateway.MessageReactionRemoveAllEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			m.Reactions = nil
			return true
		})

	case *gateway.MessageReactionRemoveEmojiEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			var i = findReaction(m.Reactions, ev.Emoji)
			if i < 0 {
				return false
			}
			m.Reactions = append(m.Reactions[:i], m.Reactions[i+1:]...)
			return true
		})

	case *gateway.PresenceUpdateEvent:
		if err := s.Cabinet.PresenceSet(ev.GuildID, ev.Presence, true); err != nil {
			s.stateErr(err, "failed to update presence in state")
		}

	case *gateway.PresencesReplaceEvent:
		for _, p := range *ev {
			if err := s.Cabinet.PresenceSet(p.GuildID, p.Presence, true); err != nil {
				s.stateErr(err, "failed to update presence in state")
			}
		}

	case *gateway.SessionsReplaceEvent:
		// TODO

	case *gateway.UserGuildSettingsUpdateEvent:
		// TODO

	case *gateway.UserSettingsUpdateEvent:
		s.readyMu.Lock()
		s.ready.UserSettings = &ev.UserSettings
		s.readyMu.Unlock()

	case *gateway.UserNoteUpdateEvent:
		// TODO

	case *gateway.UserUpdateEvent:
		if err := s.Cabinet.MyselfSet(ev.User, true); err != nil {
			s.stateErr(err, "failed to update myself from USER_UPDATE")
		}

	case *gateway.VoiceStateUpdateEvent:
		vs := &ev.VoiceState
		if vs.ChannelID == 0 {
			if err := s.Cabinet.VoiceStateRemove(vs.GuildID, vs.UserID); err != nil {
				s.stateErr(err, "failed to remove voice state from state")
			}
		} else {
			if err := s.Cabinet.VoiceStateSet(vs.GuildID, *vs, true); err != nil {
				s.stateErr(err, "failed to update voice state in state")
			}
		}
	}
}

func (s *State) stateErr(err error, wrap string) {
	s.StateLog(errors.Wrap(err, wrap))
}
func (s *State) batchLog(errors []error) {
	for _, err := range errors {
		s.StateLog(err)
	}
}

// Helper functions

func (s *State) editMessage(ch discord.ChannelID, msg discord.MessageID, fn func(m *discord.Message) bool) {
	m, err := s.Cabinet.Message(ch, msg)
	if err != nil {
		return
	}
	if !fn(m) {
		return
	}
	if err := s.Cabinet.MessageSet(*m, true); err != nil {
		s.stateErr(err, "failed to save message in reaction add")
	}
}

func findReaction(rs []discord.Reaction, emoji discord.Emoji) int {
	for i := range rs {
		if rs[i].Emoji.ID == emoji.ID && rs[i].Emoji.Name == emoji.Name {
			return i
		}
	}
	return -1
}

func storeGuildCreate(cab store.Cabinet, guild *gateway.GuildCreateEvent) []error {
	if guild.Unavailable {
		return nil
	}

	stack, errs := newErrorStack()

	if err := cab.GuildSet(guild.Guild, false); err != nil {
		errs(err, "failed to set guild in Ready")
	}

	// Handle guild emojis
	if guild.Emojis != nil {
		if err := cab.EmojiSet(guild.ID, guild.Emojis, false); err != nil {
			errs(err, "failed to set guild emojis")
		}
	}

	// Handle guild member
	for _, m := range guild.Members {
		if err := cab.MemberSet(guild.ID, m, false); err != nil {
			errs(err, "failed to set guild member in Ready")
		}
	}

	// Handle guild channels
	for _, ch := range guild.Channels {
		// I HATE Discord.
		ch.GuildID = guild.ID

		if err := cab.ChannelSet(ch, false); err != nil {
			errs(err, "failed to set guild channel in Ready")
		}
	}

	// Handle guild presences
	for _, p := range guild.Presences {
		if err := cab.PresenceSet(guild.ID, p, false); err != nil {
			errs(err, "failed to set guild presence in Ready")
		}
	}

	// Handle guild voice states
	for _, v := range guild.VoiceStates {
		if err := cab.VoiceStateSet(guild.ID, v, false); err != nil {
			errs(err, "failed to set guild voice state in Ready")
		}
	}

	return *stack
}

func newErrorStack() (*[]error, func(error, string)) {
	var errs = new([]error)
	return errs, func(err error, wrap string) {
		*errs = append(*errs, errors.Wrap(err, wrap))
	}
}
