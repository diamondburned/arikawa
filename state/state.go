// Package state provides interfaces for a local or remote state, as well as
// abstractions around the REST API and Gateway events.
package state

import (
	"context"
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/gateway/shard"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/diamondburned/arikawa/v2/state/store"
	"github.com/diamondburned/arikawa/v2/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v2/utils/handler"

	"github.com/pkg/errors"
)

var (
	MaxFetchMembers uint = 1000
	MaxFetchGuilds  uint = 100
)

// State is the cache to store events coming from Discord as well as data from
// API calls.
//
// Store
//
// The state basically provides abstractions on top of the API and the state
// storage (Store). The state storage is effectively a set of interfaces which
// allow arbitrary backends to be implemented.
//
// The default storage backend is a typical in-memory structure consisting of
// maps and slices. Custom backend implementations could embed this storage
// backend as an in-memory fallback. A good example of this would be embedding
// the default store for messages only, while handling everything else in Redis.
//
// The package also provides a no-op store (NoopStore) that implementations
// could embed. This no-op store will always return an error, which makes the
// state fetch information from the API. The setters are all no-ops, so the
// fetched data won't be updated.
//
// Handler
//
// The state uses its own handler over session's to make all handlers run after
// the state updates itself. A PreHandler is exposed in any case the user needs
// the handlers to run before the state updates itself. Refer to that field's
// documentation.
//
// The state also provides extra events and overrides to make up for Discord's
// inconsistencies in data. The following are known instances of such.
//
// The Guild Create event is split up to make the state's Guild Available, Guild
// Ready and Guild Join events. Refer to these events' documentations for more
// information.
//
// The Message Create and Message Update events with the Member field provided
// will have the User field copied from Author. This is because the User field
// will be empty, while the Member structure expects it to be there.
type State struct {
	*session.Session
	store.Cabinet

	// *: State doesn't actually keep track of pinned messages.

	readyMu *sync.Mutex
	ready   gateway.ReadyEvent

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
	// with the State.
	*handler.Handler

	// List of channels with few messages, so it doesn't bother hitting the API
	// again.
	fewMessages map[discord.ChannelID]struct{}
	fewMutex    *sync.Mutex

	// unavailableGuilds is a set of discord.GuildIDs of guilds that became
	// unavailable after connecting to the gateway, i.e. they were sent in a
	// GuildUnavailableEvent.
	unavailableGuilds map[discord.GuildID]struct{}
	// unreadyGuilds is a set of discord.GuildIDs of the guilds received during
	// the Ready event. After receiving guild create events for those guilds,
	// they will be removed.
	unreadyGuilds map[discord.GuildID]struct{}
	guildMutex    *sync.Mutex

	// NoResetOnReady prevent the state from resetting on every Ready event.
	// Shard managers should set this to true, since the sequential start of
	// shards would otherwise corrupt the state on each individual Ready event.
	NoResetOnReady bool
}

// New creates a new state.
func New(token string) (*State, error) {
	return NewWithStore(token, defaultstore.New())
}

// NewWithIntents creates a new state with the given gateway intents. For more
// information, refer to gateway.Intents.
func NewWithIntents(token string, intents ...gateway.Intents) (*State, error) {
	s, err := session.NewWithIntents(token, intents...)
	if err != nil {
		return nil, err
	}

	return newWithAutoRescale(s, defaultstore.New()), nil
}

func NewWithStore(token string, cabinet store.Cabinet) (*State, error) {
	s, err := session.New(token)
	if err != nil {
		return nil, err
	}

	return newWithAutoRescale(s, cabinet), nil
}

func newWithAutoRescale(s *session.Session, cabinet store.Cabinet) *State {
	state := NewFromSession(s, cabinet)
	state.ShardManager.Rescale = func() *shard.Manager {
		token := s.ShardManager.Gateways()[0].Identifier.Token

		m, err := shard.NewManager(token)
		if err != nil {
			return nil
		}

		state.Reset()
		return m
	}
	state.NoResetOnReady = true

	return state
}

// NewFromSession creates a new State from the passed Session and Cabinet.
func NewFromSession(s *session.Session, cabinet store.Cabinet) *State {
	state := &State{
		Session:           s,
		Cabinet:           cabinet,
		Handler:           handler.New(),
		StateLog:          func(err error) {},
		readyMu:           new(sync.Mutex),
		fewMessages:       map[discord.ChannelID]struct{}{},
		fewMutex:          new(sync.Mutex),
		unavailableGuilds: make(map[discord.GuildID]struct{}),
		unreadyGuilds:     make(map[discord.GuildID]struct{}),
		guildMutex:        new(sync.Mutex),
	}

	state.hookSession()

	return state
}

// Close closes the State's gateway connection gracefully and resets the State.
func (s *State) Close() error {
	if err := s.Session.Close(); err != nil {
		return err
	}

	return s.Reset()
}

// Reset resets the Cabinet and other internal state.
func (s *State) Reset() error {
	s.fewMutex.Lock()
	s.fewMessages = make(map[discord.ChannelID]struct{})
	s.fewMutex.Unlock()

	s.guildMutex.Lock()
	s.unavailableGuilds = make(map[discord.GuildID]struct{})
	s.unreadyGuilds = make(map[discord.GuildID]struct{})
	s.guildMutex.Unlock()

	return s.Cabinet.Reset()
}

// WithContext returns a shallow copy of State with the context replaced in the
// API client. All methods called on the State will use this given context. This
// method is thread-safe.
func (s *State) WithContext(ctx context.Context) *State {
	copied := *s
	copied.Session = s.Session.WithContext(ctx)

	return &copied
}

// Ready returns a copy of the Ready event. Although this function is safe to
// call concurrently, its values should still not be changed, as certain types
// like slices are not concurrent-safe.
//
// Note that if Ready events are not received yet, then the returned event will
// be a zero-value Ready instance.
func (s *State) Ready() gateway.ReadyEvent {
	s.readyMu.Lock()
	r := s.ready
	s.readyMu.Unlock()

	return r
}

//// Helper methods

func (s *State) AuthorDisplayName(message *gateway.MessageCreateEvent) string {
	if !message.GuildID.IsValid() {
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

func (s *State) MemberDisplayName(guildID discord.GuildID, userID discord.UserID) (string, error) {
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
	if !message.GuildID.IsValid() { // this is a dm
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

func (s *State) MemberColor(guildID discord.GuildID, userID discord.UserID) (discord.Color, error) {
	var wg sync.WaitGroup

	var (
		g *discord.Guild
		m *discord.Member

		gerr = store.ErrNotFound
		merr = store.ErrNotFound
	)

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
		g, gerr = s.Cabinet.Guild(guildID)
	}

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildMembers) {
		m, merr = s.Cabinet.Member(guildID, userID)
	}

	switch {
	case gerr != nil && merr != nil:
		wg.Add(1)
		go func() {
			g, gerr = s.fetchGuild(guildID)
			wg.Done()
		}()

		m, merr = s.fetchMember(guildID, userID)
	case gerr != nil:
		g, gerr = s.fetchGuild(guildID)
	case merr != nil:
		m, merr = s.fetchMember(guildID, userID)
	}

	wg.Wait()

	if gerr != nil {
		return 0, errors.Wrap(merr, "failed to get guild")
	}
	if merr != nil {
		return 0, errors.Wrap(merr, "failed to get member")
	}

	return discord.MemberColor(*g, *m), nil
}

////

// Permissions gets the user's permissions in the given channel. If the channel
// is not in any guild, then an error is returned.
func (s *State) Permissions(
	channelID discord.ChannelID, userID discord.UserID) (discord.Permissions, error) {

	ch, err := s.Channel(channelID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get channel")
	}

	if !ch.GuildID.IsValid() {
		return 0, errors.New("channel is not in a guild")
	}

	var wg sync.WaitGroup

	var (
		g *discord.Guild
		m *discord.Member

		gerr = store.ErrNotFound
		merr = store.ErrNotFound
	)

	if s.ShardManager.FromGuildID(ch.GuildID).HasIntents(gateway.IntentGuilds) {
		g, gerr = s.Cabinet.Guild(ch.GuildID)
	}

	if s.ShardManager.FromGuildID(ch.GuildID).HasIntents(gateway.IntentGuildMembers) {
		m, merr = s.Cabinet.Member(ch.GuildID, userID)
	}

	switch {
	case gerr != nil && merr != nil:
		wg.Add(1)
		go func() {
			g, gerr = s.fetchGuild(ch.GuildID)
			wg.Done()
		}()

		m, merr = s.fetchMember(ch.GuildID, userID)
	case gerr != nil:
		g, gerr = s.fetchGuild(ch.GuildID)
	case merr != nil:
		m, merr = s.fetchMember(ch.GuildID, userID)
	}

	wg.Wait()

	if gerr != nil {
		return 0, errors.Wrap(merr, "failed to get guild")
	}
	if merr != nil {
		return 0, errors.Wrap(merr, "failed to get member")
	}

	return discord.CalcOverwrites(*g, *ch, *m), nil
}

////

func (s *State) Me() (*discord.User, error) {
	u, err := s.Cabinet.Me()
	if err == nil {
		return u, nil
	}

	u, err = s.Session.Me()
	if err != nil {
		return nil, err
	}

	return u, s.Cabinet.MyselfSet(*u)
}

////

func (s *State) Channel(id discord.ChannelID) (c *discord.Channel, err error) {
	c, err = s.Cabinet.Channel(id)
	if err == nil && s.tracksChannel(c) {
		return
	}

	c, err = s.Session.Channel(id)
	if err != nil {
		return
	}

	if s.tracksChannel(c) {
		err = s.Cabinet.ChannelSet(*c)
	}

	return
}

func (s *State) Channels(guildID discord.GuildID) (cs []discord.Channel, err error) {
	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
		cs, err = s.Cabinet.Channels(guildID)
		if err == nil {
			return
		}
	}

	cs, err = s.Session.Channels(guildID)
	if err != nil {
		return
	}

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
		for _, c := range cs {
			if err = s.Cabinet.ChannelSet(c); err != nil {
				return
			}
		}
	}

	return
}

func (s *State) CreatePrivateChannel(recipient discord.UserID) (*discord.Channel, error) {
	c, err := s.Cabinet.CreatePrivateChannel(recipient)
	if err == nil {
		return c, nil
	}

	c, err = s.Session.CreatePrivateChannel(recipient)
	if err != nil {
		return nil, err
	}

	return c, s.Cabinet.ChannelSet(*c)
}

// PrivateChannels gets the direct messages of the user.
// This is not supported for bots.
func (s *State) PrivateChannels() ([]discord.Channel, error) {
	cs, err := s.Cabinet.PrivateChannels()
	if err == nil {
		return cs, nil
	}

	cs, err = s.Session.PrivateChannels()
	if err != nil {
		return nil, err
	}

	for _, c := range cs {
		if err := s.Cabinet.ChannelSet(c); err != nil {
			return nil, err
		}
	}

	return cs, nil
}

////

func (s *State) Emoji(
	guildID discord.GuildID, emojiID discord.EmojiID) (e *discord.Emoji, err error) {

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildEmojis) {
		e, err = s.Cabinet.Emoji(guildID, emojiID)
		if err == nil {
			return
		}
	} else { // Fast path
		return s.Session.Emoji(guildID, emojiID)
	}

	es, err := s.Session.Emojis(guildID)
	if err != nil {
		return nil, err
	}

	if err = s.Cabinet.EmojiSet(guildID, es); err != nil {
		return
	}

	for _, e := range es {
		if e.ID == emojiID {
			return &e, nil
		}
	}

	return nil, store.ErrNotFound
}

func (s *State) Emojis(guildID discord.GuildID) (es []discord.Emoji, err error) {
	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildEmojis) {
		es, err = s.Cabinet.Emojis(guildID)
		if err == nil {
			return
		}
	}

	es, err = s.Session.Emojis(guildID)
	if err != nil {
		return
	}

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildEmojis) {
		err = s.Cabinet.EmojiSet(guildID, es)
	}

	return
}

////

func (s *State) Guild(id discord.GuildID) (*discord.Guild, error) {
	if s.ShardManager.FromGuildID(id).HasIntents(gateway.IntentGuilds) {
		c, err := s.Cabinet.Guild(id)
		if err == nil {
			return c, nil
		}
	}

	return s.fetchGuild(id)
}

// Guilds will only fill a maximum of 100 guilds from the API.
func (s *State) Guilds() (gs []discord.Guild, err error) {
	hasGuildsIntent := true
	for _, g := range s.ShardManager.Gateways() {
		if !g.HasIntents(gateway.IntentGuilds) {
			hasGuildsIntent = false
			break
		}
	}

	if hasGuildsIntent {
		gs, err = s.Cabinet.Guilds()
		if err == nil {
			return gs, nil
		}
	}

	gs, err = s.Session.Guilds(MaxFetchGuilds)
	if err != nil {
		return nil, err
	}

	for _, g := range gs {
		if s.ShardManager.FromGuildID(g.ID).HasIntents(gateway.IntentGuilds) {
			if err = s.Cabinet.GuildSet(g); err != nil {
				return gs, err
			}
		}
	}

	return gs, nil
}

////

func (s *State) Member(guildID discord.GuildID, userID discord.UserID) (*discord.Member, error) {
	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildMembers) {
		m, err := s.Cabinet.Member(guildID, userID)
		if err == nil {
			return m, nil
		}
	}

	return s.fetchMember(guildID, userID)
}

func (s *State) Members(guildID discord.GuildID) (ms []discord.Member, err error) {
	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildMembers) {
		ms, err = s.Cabinet.Members(guildID)
		if err == nil {
			return
		}
	}

	ms, err = s.Session.Members(guildID, MaxFetchMembers)
	if err != nil {
		return
	}

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildMembers) {
		for _, m := range ms {
			if err = s.Cabinet.MemberSet(guildID, m); err != nil {
				return
			}
		}
	}

	return
}

////

func (s *State) Message(
	channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {

	m, err := s.Cabinet.Message(channelID, messageID)
	if err == nil && s.tracksMessage(m) {
		return m, nil
	}

	var (
		wg sync.WaitGroup

		c    *discord.Channel
		cerr = store.ErrNotFound
	)

	c, cerr = s.Cabinet.Channel(channelID)
	if cerr != nil || !s.tracksChannel(c) {
		wg.Add(1)
		go func() {
			c, cerr = s.Session.Channel(channelID)
			if cerr == nil && s.ShardManager.FromGuildID(c.GuildID).HasIntents(gateway.IntentGuilds) {
				cerr = s.Cabinet.ChannelSet(*c)
			}

			wg.Done()
		}()
	}

	m, err = s.Session.Message(channelID, messageID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch message")
	}

	wg.Wait()

	if cerr != nil {
		return nil, errors.Wrap(cerr, "unable to fetch channel")
	}

	m.ChannelID = c.ID
	m.GuildID = c.GuildID

	if s.tracksMessage(m) {
		err = s.Cabinet.MessageSet(*m)
	}

	return m, err
}

// Messages fetches maximum 100 messages from the API, if it has to. There is
// no limit if it's from the State storage.
func (s *State) Messages(channelID discord.ChannelID) ([]discord.Message, error) {
	// TODO: Think of a design that doesn't rely on MaxMessages().
	var maxMsgs = s.MaxMessages()

	ms, err := s.Cabinet.Messages(channelID)
	if err == nil && (len(ms) == 0 || s.tracksMessage(&ms[0])) {
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

	// New messages fetched weirdly does not have GuildIDs filled. We'll try and
	// get it for consistency with incoming message creates.
	var guildID discord.GuildID

	// A bit too convoluted, but whatever.
	c, err := s.Channel(channelID)
	if err == nil {
		// If it's 0, it's 0 anyway. We don't need a check here.
		guildID = c.GuildID
	}

	if len(ms) > 0 && s.tracksMessage(&ms[0]) {
		// Iterate in reverse, since the store is expected to prepend the latest
		// messages.
		for i := len(ms) - 1; i >= 0; i-- {
			// Set the guild ID, fine if it's 0 (it's already 0 anyway).
			ms[i].GuildID = guildID

			if err := s.Cabinet.MessageSet(ms[i]); err != nil {
				return nil, err
			}
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

// Presence checks the state for user presences. If no guildID is given, it
// will look for the presence in all cached guilds.
func (s *State) Presence(
	guildID discord.GuildID, userID discord.UserID) (*gateway.Presence, error) {

	if !s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildPresences) {
		return nil, store.ErrNotFound
	}

	// If there's no guild ID, look in all guilds
	if !guildID.IsValid() {
		if !s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
			return nil, store.ErrNotFound
		}

		g, err := s.Cabinet.Guilds()
		if err != nil {
			return nil, err
		}

		for _, g := range g {
			if p, err := s.Cabinet.Presence(g.ID, userID); err == nil {
				return p, nil
			}
		}

		return nil, store.ErrNotFound
	}

	return s.Cabinet.Presence(guildID, userID)
}

////

func (s *State) Role(
	guildID discord.GuildID, roleID discord.RoleID) (target *discord.Role, err error) {

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
		target, err = s.Cabinet.Role(guildID, roleID)
		if err == nil {
			return
		}
	}

	rs, err := s.Session.Roles(guildID)
	if err != nil {
		return
	}

	for _, r := range rs {
		if r.ID == roleID {
			r := r // copy to prevent mem aliasing
			target = &r
		}

		if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
			if err = s.RoleSet(guildID, r); err != nil {
				return
			}
		}
	}

	if target == nil {
		return nil, store.ErrNotFound
	}

	return
}

func (s *State) Roles(guildID discord.GuildID) ([]discord.Role, error) {
	rs, err := s.Cabinet.Roles(guildID)
	if err == nil {
		return rs, nil
	}

	rs, err = s.Session.Roles(guildID)
	if err != nil {
		return nil, err
	}

	if s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuilds) {
		for _, r := range rs {
			if err := s.RoleSet(guildID, r); err != nil {
				return rs, err
			}
		}
	}

	return rs, nil
}

func (s *State) fetchGuild(id discord.GuildID) (g *discord.Guild, err error) {
	g, err = s.Session.Guild(id)
	if err == nil && s.ShardManager.FromGuildID(id).HasIntents(gateway.IntentGuilds) {
		err = s.Cabinet.GuildSet(*g)
	}

	return
}

func (s *State) fetchMember(
	guildID discord.GuildID, userID discord.UserID) (m *discord.Member, err error) {

	m, err = s.Session.Member(guildID, userID)
	if err == nil && s.ShardManager.FromGuildID(guildID).HasIntents(gateway.IntentGuildMembers) {
		err = s.Cabinet.MemberSet(guildID, *m)
	}

	return
}

// tracksMessage reports whether the state would track the passed message and
// messages from the same channel.
func (s *State) tracksMessage(m *discord.Message) bool {
	g := s.ShardManager.FromGuildID(m.GuildID)
	return (m.GuildID.IsValid() && g.HasIntents(gateway.IntentGuildMessages)) ||
		(!m.GuildID.IsValid() && g.HasIntents(gateway.IntentDirectMessages))
}

// tracksChannel reports whether the state would track the passed channel.
func (s *State) tracksChannel(c *discord.Channel) bool {
	return (c.GuildID.IsValid() &&
		s.ShardManager.FromGuildID(c.GuildID).HasIntents(gateway.IntentGuilds)) ||
		!c.GuildID.IsValid()
}
