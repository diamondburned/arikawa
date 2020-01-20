// Package state provides interfaces for a local or remote state, as well as
// abstractions around the REST API and Gateway events.
package state

import (
	"log"
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

	// ErrorLog logs all errors that handler might have, including state fails.
	// This handler will also be used for Session, which would also be used for
	// Gateway. Defaults to log.Println.
	ErrorLog func(error)

	// PreHandler is the manual hook that is executed before the State handler
	// is. This should only be used for low-level operations.
	// It's recommended to set Synchronous to true if you mutate the events.
	PreHandler *handler.Handler // default nil

	unhooker func()

	// List of channels with few messages, so it doesn't bother hitting the API
	// again.
	fewMessages []discord.Snowflake
	fewMutex    sync.Mutex
}

func NewFromSession(s *session.Session, store Store) (*State, error) {
	state := &State{
		Session: s,
		Store:   store,
		ErrorLog: func(err error) {
			log.Println("arikawa/state error:", err)
		},
	}

	s.ErrorLog = func(err error) {
		state.ErrorLog(err)
	}

	return state, state.hookSession()
}

func New(token string) (*State, error) {
	return NewWithStore(token, NewDefaultStore(&DefaultStoreOptions{
		MaxMessages: 50,
	}))
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

////

func (s *State) Permissions(
	channelID, userID discord.Snowflake) (discord.Permissions, error) {

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

func (s *State) Self() (*discord.User, error) {
	u, err := s.Store.Self()
	if err == nil {
		return u, nil
	}

	u, err = s.Session.Me()
	if err != nil {
		return nil, err
	}

	return u, s.Store.SelfSet(u)
}

////

func (s *State) Channel(id discord.Snowflake) (*discord.Channel, error) {
	c, err := s.Store.Channel(id)
	if err == nil {
		return c, nil
	}

	c, err = s.Session.Channel(id)
	if err != nil {
		return nil, err
	}

	return c, s.Store.ChannelSet(c)
}

func (s *State) Channels(guildID discord.Snowflake) ([]discord.Channel, error) {
	c, err := s.Store.Channels(guildID)
	if err == nil {
		return c, nil
	}

	c, err = s.Session.Channels(guildID)
	if err != nil {
		return nil, err
	}

	for _, ch := range c {
		if err := s.Store.ChannelSet(&ch); err != nil {
			return nil, err
		}
	}

	return c, nil
}

////

func (s *State) Emoji(
	guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {

	e, err := s.Store.Emoji(guildID, emojiID)
	if err == nil {
		return e, nil
	}

	es, err := s.Session.Emojis(guildID)
	if err != nil {
		return nil, err
	}

	if err := s.Store.EmojiSet(guildID, es); err != nil {
		return nil, err
	}

	for _, e := range es {
		if e.ID == emojiID {
			return &e, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *State) Emojis(guildID discord.Snowflake) ([]discord.Emoji, error) {
	e, err := s.Store.Emojis(guildID)
	if err == nil {
		return e, nil
	}

	es, err := s.Session.Emojis(guildID)
	if err != nil {
		return nil, err
	}

	return es, s.Store.EmojiSet(guildID, es)
}

////

func (s *State) Guild(id discord.Snowflake) (*discord.Guild, error) {
	c, err := s.Store.Guild(id)
	if err == nil {
		return c, nil
	}

	c, err = s.Session.Guild(id)
	if err != nil {
		return nil, err
	}

	return c, s.Store.GuildSet(c)
}

// Guilds will only fill a maximum of 100 guilds from the API.
func (s *State) Guilds() ([]discord.Guild, error) {
	c, err := s.Store.Guilds()
	if err == nil {
		return c, nil
	}

	c, err = s.Session.Guilds(MaxFetchGuilds)
	if err != nil {
		return nil, err
	}

	for _, ch := range c {
		if err := s.Store.GuildSet(&ch); err != nil {
			return nil, err
		}
	}

	return c, nil
}

////

func (s *State) Member(
	guildID, userID discord.Snowflake) (*discord.Member, error) {

	m, err := s.Store.Member(guildID, userID)
	if err == nil {
		return m, nil
	}

	m, err = s.Session.Member(guildID, userID)
	if err != nil {
		return nil, err
	}

	if err := s.Store.MemberSet(guildID, m); err != nil {
		return m, err
	}

	return m, s.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildID:   []discord.Snowflake{guildID},
		Presences: true,
	})
}

func (s *State) Members(guildID discord.Snowflake) ([]discord.Member, error) {
	ms, err := s.Store.Members(guildID)
	if err == nil {
		return ms, nil
	}

	ms, err = s.Session.Members(guildID, MaxFetchMembers)
	if err != nil {
		return nil, err
	}

	for _, m := range ms {
		if err := s.Store.MemberSet(guildID, &m); err != nil {
			return nil, err
		}
	}

	return ms, s.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildID:   []discord.Snowflake{guildID},
		Presences: true,
	})
}

////

func (s *State) Message(
	channelID, messageID discord.Snowflake) (*discord.Message, error) {

	m, err := s.Store.Message(channelID, messageID)
	if err == nil {
		return m, nil
	}

	m, err = s.Session.Message(channelID, messageID)
	if err != nil {
		return nil, err
	}

	return m, s.Store.MessageSet(m)
}

// Messages fetches maximum 100 messages from the API, if it has to. There is no
// limit if it's from the State storage.
func (s *State) Messages(channelID discord.Snowflake) ([]discord.Message, error) {
	// TODO: Think of a design that doesn't rely on MaxMessages().
	var maxMsgs = s.MaxMessages()

	ms, err := s.Store.Messages(channelID)
	if err == nil {
		// If the state already has as many messages as it can, skip the API.
		if maxMsgs <= len(ms) {
			return ms, nil
		}

		// Is the channel tiny?
		s.fewMutex.Lock()
		for _, ch := range s.fewMessages {
			if ch == channelID {
				// Yes, skip the state.
				s.fewMutex.Unlock()
				return ms, nil
			}
		}

		// No, fetch from the state.
		s.fewMutex.Unlock()
	}

	ms, err = s.Session.Messages(channelID, 100)
	if err != nil {
		return nil, err
	}

	for _, m := range ms {
		if err := s.Store.MessageSet(&m); err != nil {
			return nil, err
		}
	}

	if len(ms) < maxMsgs {
		// Tiny channel, store this.
		s.fewMutex.Lock()
		s.fewMessages = append(s.fewMessages, channelID)
		s.fewMutex.Unlock()

		return ms, nil
	}

	// Since the latest messages are at the end and we already know the maxMsgs,
	// we could slice this right away.
	return ms[:maxMsgs], nil
}

////

func (s *State) Presence(
	guildID, userID discord.Snowflake) (*discord.Presence, error) {

	return s.Store.Presence(guildID, userID)
}

func (s *State) Presences(
	guildID discord.Snowflake) ([]discord.Presence, error) {

	return s.Store.Presences(guildID)
}

////

func (s *State) Role(
	guildID, roleID discord.Snowflake) (*discord.Role, error) {

	r, err := s.Store.Role(guildID, roleID)
	if err == nil {
		return r, nil
	}

	rs, err := s.Session.Roles(guildID)
	if err != nil {
		return nil, err
	}

	var role *discord.Role

	for _, r := range rs {
		if r.ID == roleID {
			role = &r
		}

		if err := s.RoleSet(guildID, &r); err != nil {
			return role, err
		}
	}

	return role, nil
}

func (s *State) Roles(guildID discord.Snowflake) ([]discord.Role, error) {
	rs, err := s.Store.Roles(guildID)
	if err == nil {
		return rs, nil
	}

	rs, err = s.Session.Roles(guildID)
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		if err := s.RoleSet(guildID, &r); err != nil {
			return rs, err
		}
	}

	return rs, nil
}
