// Package state provides interfaces for a local or remote state, as well as
// abstractions around the REST API and Gateway events.
package state

import (
	"context"
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
	fewMutex    *sync.Mutex
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

func NewFromSession(s *session.Session, store Store) (*State, error) {
	state := &State{
		Session:     s,
		Store:       store,
		Handler:     handler.New(),
		StateLog:    func(err error) {},
		fewMessages: map[discord.Snowflake]struct{}{},
		fewMutex:    new(sync.Mutex),
	}

	return state, state.hookSession()
}

// WithContext returns a shallow copy of State with the context replaced in the
// API client. All methods called on the State will use this given context. This
// method is thread-safe.
func (s *State) WithContext(ctx context.Context) *State {
	copied := *s
	copied.Client = copied.Client.WithContext(ctx)

	return &copied
}

//// Helper methods

func (s *State) AuthorDisplayName(message *gateway.MessageCreateEvent) string {
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

func (s *State) MemberDisplayName(guildID, userID discord.Snowflake) (string, error) {
	member, err := s.Member(guildID, userID)
	if err != nil {
		return "", err
	}

	if member.Nick == "" {
		return member.User.Username, nil
	}

	return member.Nick, nil
}

func (s *State) AuthorColor(message *gateway.MessageCreateEvent) (discord.Color, error) {
	if !message.GuildID.Valid() { // this is a dm
		return discord.DefaultMemberColor, nil
	}

	if message.Member != nil {
		guild, err := s.Guild(message.GuildID)
		if err != nil {
			return 0, err
		}
		return discord.MemberColor(*guild, *message.Member), nil
	}

	return s.MemberColor(message.GuildID, message.Author.ID)
}

func (s *State) MemberColor(guildID, userID discord.Snowflake) (discord.Color, error) {
	var wg sync.WaitGroup

	g, gerr := s.Store.Guild(guildID)
	if gerr != nil {
		wg.Add(1)
		go func() {
			g, gerr = s.Session.Guild(guildID)
			wg.Done()
		}()
	}

	m, merr := s.Store.Member(guildID, userID)
	if merr != nil {
		m, merr = s.Member(guildID, userID)
		if merr != nil {
			return 0, errors.Wrap(merr, "failed to get member")
		}
	}

	wg.Wait()

	if gerr != nil {
		return 0, errors.Wrap(merr, "failed to get guild")
	}

	return discord.MemberColor(*g, *m), nil
}

////

func (s *State) Permissions(channelID, userID discord.Snowflake) (discord.Permissions, error) {
	ch, err := s.Channel(channelID)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get channel")
	}

	var wg sync.WaitGroup

	g, gerr := s.Store.Guild(ch.GuildID)
	if gerr != nil {
		wg.Add(1)
		go func() {
			g, gerr = s.Session.Guild(ch.GuildID)
			wg.Done()
		}()
	}

	m, merr := s.Store.Member(ch.GuildID, userID)
	if merr != nil {
		m, merr = s.Member(ch.GuildID, userID)
		if merr != nil {
			return 0, errors.Wrap(merr, "failed to get member")
		}
	}

	wg.Wait()

	if gerr != nil {
		return 0, errors.Wrap(merr, "failed to get guild")
	}

	return discord.CalcOverwrites(*g, *ch, *m), nil
}

////

func (s *State) Me() (*discord.User, error) {
	u, err := s.Store.Me()
	if err == nil {
		return u, nil
	}

	u, err = s.Session.Me()
	if err != nil {
		return nil, err
	}

	return u, s.Store.MyselfSet(u)
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
		ch := ch

		if err := s.Store.ChannelSet(&ch); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (s *State) CreatePrivateChannel(recipient discord.Snowflake) (*discord.Channel, error) {
	c, err := s.Store.CreatePrivateChannel(recipient)
	if err == nil {
		return c, nil
	}

	c, err = s.Session.CreatePrivateChannel(recipient)
	if err != nil {
		return nil, err
	}

	return c, s.Store.ChannelSet(c)
}

func (s *State) PrivateChannels() ([]discord.Channel, error) {
	c, err := s.Store.PrivateChannels()
	if err == nil {
		return c, nil
	}

	c, err = s.Session.PrivateChannels()
	if err != nil {
		return nil, err
	}

	for _, ch := range c {
		ch := ch

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
		ch := ch

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

	return m, s.Store.MemberSet(guildID, m)
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

	// Fill the GuildID, because Discord doesn't do it for us.
	c, err := s.Channel(channelID)
	if err == nil {
		// If it's 0, it's 0 anyway. We don't need a check here.
		m.GuildID = c.GuildID
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
		if _, ok := s.fewMessages[channelID]; ok {
			s.fewMutex.Unlock()
			return ms, nil
		}

		// No, fetch from the state.
		s.fewMutex.Unlock()
	}

	ms, err = s.Session.Messages(channelID, uint(maxMsgs))
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

	// Iterate in reverse, since the store is expected to prepend the latest
	// messages.
	for i := len(ms) - 1; i >= 0; i-- {
		// Set the guild ID, fine if it's 0 (it's already 0 anyway).
		ms[i].GuildID = guildID

		if err := s.Store.MessageSet(&ms[i]); err != nil {
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
// look for the presence in all guilds.
func (s *State) Presence(guildID, userID discord.Snowflake) (*discord.Presence, error) {
	p, err := s.Store.Presence(guildID, userID)
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
			if p, err := s.Store.Presence(g.ID, userID); err == nil {
				return p, nil
			}
		}
	}

	return nil, err
}

////

func (s *State) Role(guildID, roleID discord.Snowflake) (*discord.Role, error) {

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
		r := r

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
		r := r

		if err := s.RoleSet(guildID, &r); err != nil {
			return rs, err
		}
	}

	return rs, nil
}
