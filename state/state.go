// Package state provides interfaces for a local or remote state, as well as
// abstractions around the REST API and Gateway events.
package state

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
	"github.com/diamondburned/arikawa/session"
	"github.com/pkg/errors"
)

var (
	MaxFetchMembers uint = 1000
	MaxFetchGuilds  uint = 100
)

type State struct {
	*session.Session
	Store

	// *: State doesn't actually keep track of pinned messages.

	// Ready is not updated by the state.
	Ready gateway.ReadyEvent

	// StateLog logs all errors that come from the state cache. This includes
	// not found errors. Defaults to a no-op, as state errors aren't that
	// important.
	StateLog func(error)

	// PreHandler is the manual hook that is executed before the State handler
	// is. This should only be used for low-level operations.
	// It's recommended to set Synchronous to true if you mutate the events.
	PreHandler *handler.Handler // default nil

	// Command handler with inherited methods. Ran after PreHandler. You should
	// most of the time use this instead of Session's, to avoid race conditions
	// with the State
	*handler.Handler

	unhooker func()

	// List of channels with few messages, so it doesn't bother hitting the API
	// again.
	fewMessages map[discord.Snowflake]struct{}
	fewMutex    sync.Mutex
}

// Store serves as an option that NewFromSession uses to add in stores. All
// fields are optional.
type Store struct {
	StoreMe
	StoreChannel
	StoreEmoji
	StoreGuild
	StoreMember
	StoreMessage
	StorePresence
	StoreRole
	Resetter
}

func NewFromSession(s *session.Session, store Store) (*State, error) {
	state := &State{
		Session:     s,
		Store:       store,
		Handler:     handler.New(),
		StateLog:    func(err error) {},
		fewMessages: map[discord.Snowflake]struct{}{},
	}

	return state, state.hookSession()
}

func New(token string) (*State, error) {
	return NewWithStore(token, NewDefaultStore(nil))
}

func NewWithStore(token string, store Store) (*State, error) {
	s, err := session.New(token)
	if err != nil {
		return nil, err
	}

	return NewFromSession(s, store)
}

// Unhook removes all state handlers from the session handlers.
func (s *State) Unhook() {
	s.unhooker()
}

//// Helper methods

func (s *State) AuthorDisplayName(message discord.Message) string {
	if !message.GuildID.Valid() {
		return message.Author.Username
	}

	if message.Member != nil {
		if message.Member.Nick != "" {
			return message.Member.Nick
		}
		return message.Author.Username
	}

	n, err := s.MemberDisplayName(message.GuildID, message.Author.ID)
	if err != nil {
		return message.Author.Username
	}

	return n
}

func (s *State) MemberDisplayName(
	guildID, userID discord.Snowflake) (string, error) {

	member, err := s.Member(guildID, userID)
	if err != nil {
		return "", err
	}

	if member.Nick == "" {
		return member.User.Username, nil
	}

	return member.Nick, nil
}

func (s *State) AuthorColor(message discord.Message) discord.Color {
	if !message.GuildID.Valid() {
		return discord.DefaultMemberColor
	}

	if message.Member != nil {
		guild, err := s.Guild(message.GuildID)
		if err != nil {
			return discord.DefaultMemberColor
		}
		return discord.MemberColor(*guild, *message.Member)
	}

	return s.MemberColor(message.GuildID, message.Author.ID)
}

func (s *State) MemberColor(guildID, userID discord.Snowflake) discord.Color {
	member, err := s.Member(guildID, userID)
	if err != nil {
		return discord.DefaultMemberColor
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return discord.DefaultMemberColor
	}

	return discord.MemberColor(*guild, *member)
}

////

func (s *State) Permissions(channelID, userID discord.Snowflake) (discord.Permissions, error) {
	ch, err := s.Channel(channelID)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get channel")
	}

	g, err := s.Guild(ch.GuildID)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get guild")
	}

	m, err := s.Member(ch.GuildID, userID)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get member")
	}

	return discord.CalcOverwrites(*g, *ch, *m), nil
}

////

func (s *State) Me() (*discord.User, error) {
	if s.StoreMe != nil {
		u, err := s.StoreMe.Me()
		if err == nil {
			return u, nil
		}
	}

	u, err := s.Session.Me()
	if err != nil {
		return nil, err
	}

	if s.StoreMe != nil {
		return u, s.StoreMe.MyselfSet(u)
	}

	return u, nil
}

////

func (s *State) Channel(id discord.Snowflake) (*discord.Channel, error) {
	if s.StoreChannel != nil {
		c, err := s.StoreChannel.Channel(id)
		if err == nil {
			return c, nil
		}
	}

	c, err := s.Session.Channel(id)
	if err != nil {
		return nil, err
	}

	if s.StoreChannel != nil {
		return c, s.StoreChannel.ChannelSet(c)
	}

	return c, nil
}

func (s *State) Channels(guildID discord.Snowflake) ([]discord.Channel, error) {
	if s.StoreChannel != nil {
		c, err := s.StoreChannel.Channels(guildID)
		if err == nil {
			return c, nil
		}
	}

	c, err := s.Session.Channels(guildID)
	if err != nil {
		return nil, err
	}

	if s.StoreChannel != nil {
		for _, ch := range c {
			ch := ch

			if err := s.StoreChannel.ChannelSet(&ch); err != nil {
				return nil, err
			}
		}
	}

	return c, nil
}

////

func (s *State) Emoji(guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {
	if s.StoreEmoji != nil {
		e, err := s.StoreEmoji.Emoji(guildID, emojiID)
		if err == nil {
			return e, nil
		}
	}

	es, err := s.Session.Emojis(guildID)
	if err != nil {
		return nil, err
	}

	if s.StoreEmoji != nil {
		if err := s.StoreEmoji.EmojiSet(guildID, es); err != nil {
			return nil, err
		}
	}

	for _, e := range es {
		if e.ID == emojiID {
			return &e, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *State) Emojis(guildID discord.Snowflake) ([]discord.Emoji, error) {
	if s.StoreEmoji != nil {
		e, err := s.StoreEmoji.Emojis(guildID)
		if err == nil {
			return e, nil
		}
	}

	es, err := s.Session.Emojis(guildID)
	if err != nil {
		return nil, err
	}

	if s.StoreEmoji != nil {
		return es, s.StoreEmoji.EmojiSet(guildID, es)
	}

	return es, nil
}

////

func (s *State) Guild(id discord.Snowflake) (*discord.Guild, error) {
	if s.StoreGuild != nil {
		c, err := s.StoreGuild.Guild(id)
		if err == nil {
			return c, nil
		}
	}

	c, err := s.Session.Guild(id)
	if err != nil {
		return nil, err
	}

	if s.StoreGuild != nil {
		return c, s.StoreGuild.GuildSet(c)
	}

	return c, nil
}

// Guilds will only fill a maximum of 100 guilds from the API.
func (s *State) Guilds() ([]discord.Guild, error) {
	if s.StoreGuild != nil {
		c, err := s.StoreGuild.Guilds()
		if err == nil {
			return c, nil
		}
	}

	c, err := s.Session.Guilds(MaxFetchGuilds)
	if err != nil {
		return nil, err
	}

	if s.StoreGuild != nil {
		for i := range c {
			if err := s.StoreGuild.GuildSet(&c[i]); err != nil {
				return nil, err
			}
		}
	}

	return c, nil
}

////

func (s *State) Member(guildID, userID discord.Snowflake) (*discord.Member, error) {
	if s.StoreMember != nil {
		m, err := s.StoreMember.Member(guildID, userID)
		if err == nil {
			return m, nil
		}
	}

	m, err := s.Session.Member(guildID, userID)
	if err != nil {
		return nil, err
	}

	if s.StoreMember != nil {
		return m, s.StoreMember.MemberSet(guildID, m)
	}

	return m, nil
}

// Members when called for its first time may not return a lot.
func (s *State) Members(guildID discord.Snowflake) ([]discord.Member, error) {
	if s.StoreMember != nil {
		ms, err := s.StoreMember.Members(guildID)
		if err == nil {
			return ms, nil
		}
	}

	ms, err := s.Session.Members(guildID, MaxFetchMembers)
	if err != nil {
		return nil, err
	}

	if s.StoreMember != nil {
		for _, m := range ms {
			if err := s.StoreMember.MemberSet(guildID, &m); err != nil {
				return nil, err
			}
		}

		// idk why I wrote this
		return ms, s.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
			GuildID:   []discord.Snowflake{guildID},
			Presences: true,
		})
	}

	return ms, nil
}

////

func (s *State) Message(channelID, messageID discord.Snowflake) (*discord.Message, error) {
	if s.StoreMessage != nil {
		m, err := s.StoreMessage.Message(channelID, messageID)
		if err == nil {
			return m, nil
		}
	}

	m, err := s.Session.Message(channelID, messageID)
	if err != nil {
		return nil, err
	}

	// Fill the GuildID, because Discord doesn't do it for us.
	c, err := s.Channel(channelID)
	if err == nil {
		// If it's 0, it's 0 anyway. We don't need a check here.
		m.GuildID = c.GuildID
	}

	if s.StoreMessage != nil {
		return m, s.StoreMessage.MessageSet(m)
	}

	return m, nil
}

// Messages fetches maximum 100 messages from the API, if it has to, or it will
// use the limit from the Message State.
func (s *State) Messages(channelID discord.Snowflake) ([]discord.Message, error) {
	var maxMsgs = 100

	if s.StoreMessage != nil {
		// TODO: Think of a design that doesn't rely on MaxMessages().
		maxMsgs = s.StoreMessage.MaxMessages()

		ms, err := s.StoreMessage.Messages(channelID)
		if err == nil {
			// If the state already has as many messages as it can, skip the API.
			if maxMsgs <= len(ms) {
				return ms, nil
			}

			// Is the channel tiny?
			s.fewMutex.Lock()
			if _, ok := s.fewMessages[channelID]; ok {
				s.fewMutex.Unlock()
				return ms, nil
			}

			// No, fetch from the state.
			s.fewMutex.Unlock()
		}
	}

	ms, err := s.Session.Messages(channelID, uint(maxMsgs))
	if err != nil {
		return nil, err
	}

	// New messages fetched weirdly does not have GuildID filled. We'll try and
	// get it for consistency with incoming message creates.
	var guildID discord.Snowflake

	// A bit too convoluted, but whatever.
	c, err := s.Channel(channelID)
	if err == nil {
		// If it's 0, it's 0 anyway. We don't need a check here.
		guildID = c.GuildID
	}

	if s.StoreMessage == nil {
		return ms, nil
	}

	for i := range ms {
		// Set the guild ID, fine if it's 0 (it's already 0 anyway).
		ms[i].GuildID = guildID

		if err := s.StoreMessage.MessageSet(&ms[i]); err != nil {
			return nil, err
		}
	}

	if len(ms) < maxMsgs {
		// Tiny channel, store this.
		s.fewMutex.Lock()
		s.fewMessages[channelID] = struct{}{}
		s.fewMutex.Unlock()

		return ms, nil
	}

	// Since the latest messages are at the end and we already know the maxMsgs,
	// we could slice this right away.
	return ms[:maxMsgs], nil
}

////

// Presence checks the state for user presences. If no guildID is given, it will
// look for the presence in all guilds. The function will error out if
// StorePresences is nil.
func (s *State) Presence(guildID, userID discord.Snowflake) (*discord.Presence, error) {
	if s.StorePresence == nil {
		return nil, ErrStoreNotFound
	}

	p, err := s.StorePresence.Presence(guildID, userID)
	if err == nil {
		return p, nil
	}

	// If there's no guild ID, look in all guilds
	if !guildID.Valid() {
		g, err := s.Guilds()
		if err != nil {
			return nil, err
		}

		for _, g := range g {
			if p, err := s.StorePresence.Presence(g.ID, userID); err == nil {
				return p, nil
			}
		}
	}

	return nil, err
}

// Presences only returns presences if StorePresences is not nil.
func (s *State) Presences(guildID discord.Snowflake) ([]discord.Presence, error) {
	if s.StorePresence == nil {
		return nil, ErrStoreNotFound
	}
	return s.StorePresence.Presences(guildID)
}

////

func (s *State) Role(guildID, roleID discord.Snowflake) (*discord.Role, error) {
	if s.StoreRole != nil {
		r, err := s.StoreRole.Role(guildID, roleID)
		if err == nil {
			return r, nil
		}
	}

	rs, err := s.Session.Roles(guildID)
	if err != nil {
		return nil, err
	}

	var role *discord.Role

	for _, r := range rs {
		r := r

		if r.ID == roleID {
			role = &r
		}

		if s.StoreRole != nil {
			if err := s.StoreRole.RoleSet(guildID, &r); err != nil {
				return nil, err
			}
		}
	}

	if role == nil {
		return nil, ErrStoreNotFound
	}

	return role, nil
}

func (s *State) Roles(guildID discord.Snowflake) ([]discord.Role, error) {
	if s.StoreRole != nil {
		rs, err := s.StoreRole.Roles(guildID)
		if err == nil {
			return rs, nil
		}
	}

	rs, err := s.Session.Roles(guildID)
	if err != nil {
		return nil, err
	}

	if s.StoreRole != nil {
		for _, r := range rs {
			r := r

			if err := s.StoreRole.RoleSet(guildID, &r); err != nil {
				return rs, err
			}
		}
	}

	return rs, nil
}
