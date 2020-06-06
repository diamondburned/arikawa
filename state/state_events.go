package state

import (
	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

func (s *State) hookSession() error {
	s.unhooker = s.Session.AddHandler(func(iface interface{}) {
		s.onEvent(iface)

		switch e := iface.(type) {
		case *gateway.ReadyEvent:
			s.handleReady(e)
		case *gateway.GuildCreateEvent:
			s.handleGuildCreate(e)
		case *gateway.GuildDeleteEvent:
			s.handleGuildDelete(e)
		default:
			s.handleEvent(iface)
		}
	})

	return nil
}

func (s *State) onEvent(iface interface{}) {
	switch ev := iface.(type) {
	case *gateway.ReadyEvent:
		// Set Ready to the state
		s.Ready = *ev

		// Handle presences
		for _, p := range ev.Presences {
			p := p

			if err := s.Store.PresenceSet(0, &p); err != nil {
				s.stateErr(err, "failed to set global presence")
			}
		}

		// Handle guilds
		for i := range ev.Guilds {
			s.batchLog(storeGuildCreate(s.Store, &ev.Guilds[i])...)
		}

		// Handle private channels
		for i := range ev.PrivateChannels {
			if err := s.Store.ChannelSet(&ev.PrivateChannels[i]); err != nil {
				s.stateErr(err, "failed to set channel in state")
			}
		}

		// Handle user
		if err := s.Store.MyselfSet(&ev.User); err != nil {
			s.stateErr(err, "failed to set self in state")
		}

	case *gateway.GuildUpdateEvent:
		if err := s.Store.GuildSet((*discord.Guild)(ev)); err != nil {
			s.stateErr(err, "failed to update guild in state")
		}

	case *gateway.GuildDeleteEvent:
		if err := s.Store.GuildRemove(ev.ID); err != nil && !ev.Unavailable {
			s.stateErr(err, "failed to delete guild in state")
		}

	case *gateway.GuildMemberAddEvent:
		if err := s.Store.MemberSet(ev.GuildID, &ev.Member); err != nil {
			s.stateErr(err, "failed to add a member in state")
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
			s.stateErr(err, "failed to update a member in state")
		}

	case *gateway.GuildMemberRemoveEvent:
		if err := s.Store.MemberRemove(ev.GuildID, ev.User.ID); err != nil {
			s.stateErr(err, "failed to remove a member in state")
		}

	case *gateway.GuildMembersChunkEvent:
		for _, m := range ev.Members {
			m := m

			if err := s.Store.MemberSet(ev.GuildID, &m); err != nil {
				s.stateErr(err, "failed to add a member from chunk in state")
			}
		}

		for _, p := range ev.Presences {
			p := p

			if err := s.Store.PresenceSet(ev.GuildID, &p); err != nil {
				s.stateErr(err, "failed to add a presence from chunk in state")
			}
		}

	case *gateway.GuildRoleCreateEvent:
		if err := s.Store.RoleSet(ev.GuildID, &ev.Role); err != nil {
			s.stateErr(err, "failed to add a role in state")
		}

	case *gateway.GuildRoleUpdateEvent:
		if err := s.Store.RoleSet(ev.GuildID, &ev.Role); err != nil {
			s.stateErr(err, "failed to update a role in state")
		}

	case *gateway.GuildRoleDeleteEvent:
		if err := s.Store.RoleRemove(ev.GuildID, ev.RoleID); err != nil {
			s.stateErr(err, "failed to remove a role in state")
		}

	case *gateway.GuildEmojisUpdateEvent:
		if err := s.Store.EmojiSet(ev.GuildID, ev.Emojis); err != nil {
			s.stateErr(err, "failed to update emojis in state")
		}

	case *gateway.ChannelCreateEvent:
		if err := s.Store.ChannelSet((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "failed to create a channel in state")
		}

	case *gateway.ChannelUpdateEvent:
		if err := s.Store.ChannelSet((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "failed to update a channel in state")
		}

	case *gateway.ChannelDeleteEvent:
		if err := s.Store.ChannelRemove((*discord.Channel)(ev)); err != nil {
			s.stateErr(err, "failed to remove a channel in state")
		}

	case *gateway.ChannelPinsUpdateEvent:
		// not tracked.

	case *gateway.MessageCreateEvent:
		if err := s.Store.MessageSet(&ev.Message); err != nil {
			s.stateErr(err, "failed to add a message in state")
		}

	case *gateway.MessageUpdateEvent:
		if err := s.Store.MessageSet(&ev.Message); err != nil {
			s.stateErr(err, "failed to update a message in state")
		}

	case *gateway.MessageDeleteEvent:
		if err := s.Store.MessageRemove(ev.ChannelID, ev.ID); err != nil {
			s.stateErr(err, "failed to delete a message in state")
		}

	case *gateway.MessageDeleteBulkEvent:
		for _, id := range ev.IDs {
			if err := s.Store.MessageRemove(ev.ChannelID, id); err != nil {
				s.stateErr(err, "failed to delete bulk meessages in state")
			}
		}

	case *gateway.MessageReactionAddEvent:
		s.editMessage(ev.ChannelID, ev.MessageID, func(m *discord.Message) bool {
			if i := findReaction(m.Reactions, ev.Emoji); i > -1 {
				m.Reactions[i].Count++
			} else {
				var me bool
				if u, _ := s.Store.Me(); u != nil {
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
		if err := s.Store.PresenceSet(ev.GuildID, &ev.Presence); err != nil {
			s.stateErr(err, "failed to update presence in state")
		}

	case *gateway.PresencesReplaceEvent:
		for i := range *ev {
			p := (*ev)[i]

			if err := s.Store.PresenceSet(p.GuildID, &p); err != nil {
				s.stateErr(err, "failed to update presence in state")
			}
		}

	case *gateway.SessionsReplaceEvent:

	case *gateway.UserGuildSettingsUpdateEvent:
		for i, ugs := range s.Ready.UserGuildSettings {
			if ugs.GuildID == ev.GuildID {
				s.Ready.UserGuildSettings[i] = ev.UserGuildSettings
			}
		}

	case *gateway.UserSettingsUpdateEvent:
		s.Ready.Settings = &ev.UserSettings

	case *gateway.UserNoteUpdateEvent:
		s.Ready.Notes[ev.ID] = ev.Note

	case *gateway.UserUpdateEvent:
		if err := s.Store.MyselfSet(&ev.User); err != nil {
			s.stateErr(err, "failed to update myself from USER_UPDATE")
		}

	case *gateway.VoiceStateUpdateEvent:
		vs := &ev.VoiceState
		if vs.ChannelID == 0 {
			if err := s.Store.VoiceStateRemove(vs.GuildID, vs.UserID); err != nil {
				s.stateErr(err, "failed to remove voice state from state")
			}
		} else {
			if err := s.Store.VoiceStateSet(vs.GuildID, vs); err != nil {
				s.stateErr(err, "failed to update voice state in state")
			}
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

func storeGuildCreate(store Store, guild *gateway.GuildCreateEvent) []error {
	if guild.Unavailable {
		return nil
	}

	stack, errs := newErrorStack()

	if err := store.GuildSet(&guild.Guild); err != nil {
		errs(err, "failed to set guild in Ready")
	}

	// Handle guild emojis
	if guild.Emojis != nil {
		if err := store.EmojiSet(guild.ID, guild.Emojis); err != nil {
			errs(err, "failed to set guild emojis")
		}
	}

	// Handle guild member
	for i := range guild.Members {
		if err := store.MemberSet(guild.ID, &guild.Members[i]); err != nil {
			errs(err, "failed to set guild member in Ready")
		}
	}

	// Handle guild channels
	for i := range guild.Channels {
		// I HATE Discord.
		ch := guild.Channels[i]
		ch.GuildID = guild.ID

		if err := store.ChannelSet(&ch); err != nil {
			errs(err, "failed to set guild channel in Ready")
		}
	}

	// Handle guild presences
	for i := range guild.Presences {
		if err := store.PresenceSet(guild.ID, &guild.Presences[i]); err != nil {
			errs(err, "failed to set guild presence in Ready")
		}
	}

	// Handle guild voice states
	for i := range guild.VoiceStates {
		if err := store.VoiceStateSet(guild.ID, &guild.VoiceStates[i]); err != nil {
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
