package state

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/pkg/errors"
)

func (s *State) hookSession() error {
	s.unhooker = s.Session.AddHandler(func(iface interface{}) {
		if s.PreHandler != nil {
			s.PreHandler.Call(iface)
		}
		s.onEvent(iface)
		s.Handler.Call(iface)
	})

	return nil
}

func (s *State) onEvent(iface interface{}) {
	// TODO: voice states

	switch ev := iface.(type) {
	case *gateway.ReadyEvent:
		// Set Ready to the state
		s.Ready = *ev

		// Handle presences
		for _, p := range ev.Presences {
			p := p

			if err := s.Store.PresenceSet(0, &p); err != nil {
				s.stateErr(err, "Failed to set global presence")
			}
		}

		// Handle guilds
		for i := range ev.Guilds {
			s.batchLog(handleGuildCreate(s.Store, &ev.Guilds[i])...)
		}

		// Handle private channels
		for i := range ev.PrivateChannels {
			if err := s.Store.ChannelSet(&ev.PrivateChannels[i]); err != nil {
				s.stateErr(err, "Failed to set channel in state")
			}
		}

		// Handle user
		if err := s.Store.MyselfSet(&ev.User); err != nil {
			s.stateErr(err, "Failed to set self in state")
		}

	case *gateway.GuildCreateEvent:
		s.batchLog(handleGuildCreate(s.Store, ev)...)

	case *gateway.GuildUpdateEvent:
		if err := s.Store.GuildSet((*discord.Guild)(ev)); err != nil {
			s.stateErr(err, "Failed to update guild in state")
		}
	case *gateway.GuildDeleteEvent:
		if err := s.Store.GuildRemove(ev.ID); err != nil {
			s.stateErr(err, "Failed to delete guild in state")
		}

	case *gateway.GuildMemberAddEvent:
		if err := s.Store.MemberSet(ev.GuildID, &ev.Member); err != nil {
			s.stateErr(err, "Failed to add a member in state")
		}
	case *gateway.GuildMemberUpdateEvent:
		m, err := s.Store.Member(ev.GuildID, ev.User.ID)
		if err != nil {
			// We can't do much here.
			m = &discord.Member{}
		}

		// Update available fields from ev into m
		ev.Update(m)

		if err := s.Store.MemberSet(ev.GuildID, m); err != nil {
			s.stateErr(err, "Failed to update a member in state")
		}
	case *gateway.GuildMemberRemoveEvent:
		if err := s.Store.MemberRemove(ev.GuildID, ev.User.ID); err != nil {
			s.stateErr(err, "Failed to remove a member in state")
		}

	case *gateway.GuildMembersChunkEvent:
		for _, m := range ev.Members {
			m := m

			if err := s.Store.MemberSet(ev.GuildID, &m); err != nil {
				s.stateErr(err, "Failed to add a member from chunk in state")
			}
		}

		for _, p := range ev.Presences {
			p := p

			if err := s.Store.PresenceSet(ev.GuildID, &p); err != nil {
				s.stateErr(err, "Failed to add a presence from chunk in state")
			}
		}

	case *gateway.GuildRoleCreateEvent:
		if err := s.Store.RoleSet(ev.GuildID, &ev.Role); err != nil {
			s.stateErr(err, "Failed to add a role in state")
		}
	case *gateway.GuildRoleUpdateEvent:
		if err := s.Store.RoleSet(ev.GuildID, &ev.Role); err != nil {
			s.stateErr(err, "Failed to update a role in state")
		}
	case *gateway.GuildRoleDeleteEvent:
		if err := s.Store.RoleRemove(ev.GuildID, ev.RoleID); err != nil {
			s.stateErr(err, "Failed to remove a role in state")
		}

	case *gateway.GuildEmojisUpdateEvent:
		if err := s.Store.EmojiSet(ev.GuildID, ev.Emojis); err != nil {
			s.stateErr(err, "Failed to update emojis in state")
		}

	case *gateway.ChannelCreateEvent:
		if err := s.Store.ChannelSet((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "Failed to create a channel in state")
		}
	case *gateway.ChannelUpdateEvent:
		if err := s.Store.ChannelSet((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "Failed to update a channel in state")
		}
	case *gateway.ChannelDeleteEvent:
		if err := s.Store.ChannelRemove((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "Failed to remove a channel in state")
		}

	case *gateway.ChannelPinsUpdateEvent:
		// not tracked.

	case *gateway.MessageCreateEvent:
		if err := s.Store.MessageSet(&ev.Message); err != nil {
			s.stateErr(err, "Failed to add a message in state")
		}
	case *gateway.MessageUpdateEvent:
		if err := s.Store.MessageSet(&ev.Message); err != nil {
			s.stateErr(err, "Failed to update a message in state")
		}
	case *gateway.MessageDeleteEvent:
		if err := s.Store.MessageRemove(ev.ChannelID, ev.ID); err != nil {
			s.stateErr(err, "Failed to delete a message in state")
		}
	case *gateway.MessageDeleteBulkEvent:
		for _, id := range ev.IDs {
			if err := s.Store.MessageRemove(ev.ChannelID, id); err != nil {
				s.stateErr(err, "Failed to delete bulk meessages in state")
			}
		}
	case *gateway.MessageReactionAddEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			if i := findReaction(m.Reactions, ev.Emoji); i > -1 {
				m.Reactions[i].Count++
			} else {
				u, err := s.Store.Me()
				if err != nil {
					s.stateErr(err, "Failed to get self for reaction add")
					return false
				}
				m.Reactions = append(m.Reactions, discord.Reaction{
					Count: 1,
					Me:    ev.UserID == u.ID,
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
				u, err := s.Store.Me()
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
	case *gateway.MessageReactionRemoveEmoji:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			var i = findReaction(m.Reactions, ev.Emoji)
			if i < 0 {
				return false
			}
			m.Reactions = append(m.Reactions[:i], m.Reactions[i+1:]...)
			return true
		})

	case *gateway.PresenceUpdateEvent:
		presence := (*discord.Presence)(ev)
		if err := s.Store.PresenceSet(ev.GuildID, presence); err != nil {
			s.stateErr(err, "Failed to update presence in state")
		}

	case *gateway.PresencesReplaceEvent:
		for i := range *ev {
			p := (*ev)[i]

			if err := s.Store.PresenceSet(p.GuildID, &p); err != nil {
				s.stateErr(err, "Failed to update presence in state")
			}
		}

	case *gateway.UserGuildSettingsUpdateEvent:
		for i, ugs := range s.Ready.UserGuildSettings {
			if ugs.GuildID == ev.GuildID {
				s.Ready.UserGuildSettings[i] = gateway.UserGuildSettings(*ev)
			}
		}

	case *gateway.UserSettingsUpdateEvent:
		s.Ready.Settings = (*gateway.UserSettings)(ev)

	case *gateway.UserNoteUpdateEvent:
		s.Ready.Notes[ev.ID] = ev.Note

	case *gateway.UserUpdateEvent:
		if err := s.Store.MyselfSet((*discord.User)(ev)); err != nil {
			s.stateErr(err, "Failed to update myself from USER_UPDATE")
		}
	}
}

func (s *State) stateErr(err error, wrap string) {
	s.StateLog(errors.Wrap(err, wrap))
}
func (s *State) batchLog(errors ...error) {
	for _, err := range errors {
		s.StateLog(err)
	}
}

// Helper functions

func (s *State) editMessage(ch, msg discord.Snowflake, fn func(m *discord.Message) bool) {
	m, err := s.Store.Message(ch, msg)
	if err != nil {
		return
	}
	if !fn(m) {
		return
	}
	if err := s.Store.MessageSet(m); err != nil {
		s.stateErr(err, "Failed to save message in reaction add")
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

func handleGuildCreate(store Store, guild *gateway.GuildCreateEvent) []error {
	// If a guild is unavailable, don't populate it in the state, as the guild
	// data is very incomplete.
	if guild.Unavailable {
		return nil
	}

	stack, error := newErrorStack()

	if err := store.GuildSet(&guild.Guild); err != nil {
		error(err, "Failed to set guild in Ready")
	}

	// Handle guild emojis
	if guild.Emojis != nil {
		if err := store.EmojiSet(guild.ID, guild.Emojis); err != nil {
			error(err, "Failed to set guild emojis")
		}
	}

	// Handle guild member
	for i := range guild.Members {
		if err := store.MemberSet(guild.ID, &guild.Members[i]); err != nil {
			error(err, "Failed to set guild member in Ready")
		}
	}

	// Handle guild channels
	for i := range guild.Channels {
		if err := store.ChannelSet(&guild.Channels[i]); err != nil {
			error(err, "Failed to set guild channel in Ready")
		}
	}

	// Handle guild presences
	for i := range guild.Presences {
		if err := store.PresenceSet(guild.ID, &guild.Presences[i]); err != nil {
			error(err, "Failed to set guild presence in Ready")
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
