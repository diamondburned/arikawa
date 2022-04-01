// Package state provides interfaces for a local or remote state, as well as
// abstractions around the REST API and Gateway events.
package state

import (
	"context"
	"sync"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state/store"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/handler"

	"github.com/pkg/errors"
)

var (
	MaxFetchMembers uint = 1000
	MaxFetchGuilds  uint = 100
)

// NewShardFunc creates a shard constructor with its own state registry and
// handlers. The given opts function is called everytime the State is created.
// The user should initialize handlers and intents in the opts function.
func NewShardFunc(opts func(*shard.Manager, *State)) shard.NewShardFunc {
	return func(m *shard.Manager, id *gateway.Identifier) (shard.Shard, error) {
		sessn := session.NewCustom(*id, api.NewClient(id.Token), handler.New())
		state := NewFromSession(sessn, defaultstore.New())
		opts(m, state)
		return state, nil
	}
}

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
	*store.Cabinet

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
}

// New creates a new state.
func New(token string) *State {
	return NewWithStore(token, defaultstore.New())
}

// NewWithIntents creates a new state with the given gateway intents. For more
// information, refer to gateway.Intents.
func NewWithIntents(token string, intents ...gateway.Intents) *State {
	s := session.NewWithIntents(token, intents...)
	return NewFromSession(s, defaultstore.New())
}

// NewWithIdentifier creates a new state with the given gateway identifier.
func NewWithIdentifier(id gateway.Identifier) *State {
	s := session.NewWithIdentifier(id)
	return NewFromSession(s, defaultstore.New())
}

// NewWithStore creates a new state with the given store cabinet.
func NewWithStore(token string, cabinet *store.Cabinet) *State {
	s := session.New(token)
	return NewFromSession(s, cabinet)
}

// NewFromSession creates a new State from the passed Session and Cabinet.
func NewFromSession(s *session.Session, cabinet *store.Cabinet) *State {
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

// AuthorColor is a variant of MemberColor that possibly uses the existing
// Member field inside MessageCreateEvent.
func (s *State) AuthorColor(message *gateway.MessageCreateEvent) (discord.Color, bool) {
	if !message.GuildID.IsValid() { // this is a dm
		return discord.NullColor, false
	}

	if message.Member != nil {
		return MemberColor(message.Member, func(id discord.RoleID) *discord.Role {
			r, _ := s.Role(message.GuildID, id)
			return r
		})
	}

	return s.MemberColor(message.GuildID, message.Author.ID)
}

// MemberColor fetches the color of the member with the given user ID inside the
// guild with the given ID.
func (s *State) MemberColor(guildID discord.GuildID, userID discord.UserID) (discord.Color, bool) {
	m, err := s.Member(guildID, userID)
	if err != nil {
		return discord.NullColor, false
	}

	return MemberColor(m, func(id discord.RoleID) *discord.Role {
		r, _ := s.Role(guildID, id)
		return r
	})
}

// MemberColor is a weird variant of State's MemberColor method that allows a
// custom Role getter. If m is nil, then NullColor is returned.
func MemberColor(m *discord.Member, role func(discord.RoleID) *discord.Role) (discord.Color, bool) {
	c := discord.NullColor
	pos := -1

	if m == nil {
		return c, false
	}

	for _, roleID := range m.RoleIDs {
		if r := role(roleID); r != nil {
			if r.Color > 0 && r.Position > pos {
				c = r.Color
				pos = r.Position
			}
		}
	}

	return c, pos != -1
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

	if s.HasIntents(gateway.IntentGuilds) {
		g, gerr = s.Cabinet.Guild(ch.GuildID)
	}

	if s.HasIntents(gateway.IntentGuildMembers) {
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

	s.Cabinet.MyselfSet(*u, false)

	return u, nil
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
		s.Cabinet.ChannelSet(c, false)
	}

	return
}

func (s *State) Channels(guildID discord.GuildID) (cs []discord.Channel, err error) {
	if s.HasIntents(gateway.IntentGuilds) {
		cs, err = s.Cabinet.Channels(guildID)
		if err == nil {
			return
		}
	}

	cs, err = s.Session.Channels(guildID)
	if err != nil {
		return
	}

	if s.HasIntents(gateway.IntentGuilds) {
		for i := range cs {
			s.Cabinet.ChannelSet(&cs[i], false)
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

	s.Cabinet.ChannelSet(c, false)

	return c, nil
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

	for i := range cs {
		s.Cabinet.ChannelSet(&cs[i], false)
	}

	return cs, nil
}

////

func (s *State) Emoji(
	guildID discord.GuildID, emojiID discord.EmojiID) (e *discord.Emoji, err error) {

	if s.HasIntents(gateway.IntentGuildEmojis) {
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

	s.Cabinet.EmojiSet(guildID, es, false)

	for _, e := range es {
		if e.ID == emojiID {
			return &e, nil
		}
	}

	return nil, store.ErrNotFound
}

func (s *State) Emojis(guildID discord.GuildID) (es []discord.Emoji, err error) {
	if s.HasIntents(gateway.IntentGuildEmojis) {
		es, err = s.Cabinet.Emojis(guildID)
		if err == nil {
			return
		}
	}

	es, err = s.Session.Emojis(guildID)
	if err != nil {
		return
	}

	if s.HasIntents(gateway.IntentGuildEmojis) {
		s.Cabinet.EmojiSet(guildID, es, false)
	}

	return
}

////

func (s *State) Guild(id discord.GuildID) (*discord.Guild, error) {
	if s.HasIntents(gateway.IntentGuilds) {
		c, err := s.Cabinet.Guild(id)
		if err == nil {
			return c, nil
		}
	}

	return s.fetchGuild(id)
}

// Guilds will only fill a maximum of 100 guilds from the API.
func (s *State) Guilds() (gs []discord.Guild, err error) {
	if s.HasIntents(gateway.IntentGuilds) {
		gs, err = s.Cabinet.Guilds()
		if err == nil {
			return
		}
	}

	gs, err = s.Session.Guilds(MaxFetchGuilds)
	if err != nil {
		return
	}

	if s.HasIntents(gateway.IntentGuilds) {
		for i := range gs {
			s.Cabinet.GuildSet(&gs[i], false)
		}
	}

	return
}

////

func (s *State) Member(guildID discord.GuildID, userID discord.UserID) (*discord.Member, error) {
	if s.HasIntents(gateway.IntentGuildMembers) {
		m, err := s.Cabinet.Member(guildID, userID)
		if err == nil {
			return m, nil
		}
	}

	return s.fetchMember(guildID, userID)
}

func (s *State) Members(guildID discord.GuildID) (ms []discord.Member, err error) {
	if s.HasIntents(gateway.IntentGuildMembers) {
		ms, err = s.Cabinet.Members(guildID)
		if err == nil {
			return
		}
	}

	ms, err = s.Session.Members(guildID, MaxFetchMembers)
	if err != nil {
		return
	}

	if s.HasIntents(gateway.IntentGuildMembers) {
		for i := range ms {
			s.Cabinet.MemberSet(guildID, &ms[i], false)
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
			if cerr == nil && s.HasIntents(gateway.IntentGuilds) {
				s.Cabinet.ChannelSet(c, false)
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

	return m, err
}

// Messages returns a slice filled with the most recent messages sent in the
// channel with the passed ID. The method automatically paginates until it
// reaches the passed limit, or, if the limit is set to 0, has fetched all
// messages in the channel.
//
// As the underlying endpoint is capped at a maximum of 100 messages per
// request, at maximum a total of limit/100 rounded up requests will be made,
// although they may be less, if no more messages are available or there are
// cached messages.
// When fetching the messages, those with the highest ID, will be fetched
// first. The returned slice will be sorted from latest to oldest.
func (s *State) Messages(channelID discord.ChannelID, limit uint) ([]discord.Message, error) {
	storeMessages, err := s.Cabinet.Messages(channelID)
	if err == nil && s.tracksMessage(&storeMessages[0]) {
		// Is the channel tiny?
		s.fewMutex.Lock()
		if _, ok := s.fewMessages[channelID]; ok {
			s.fewMutex.Unlock()
			return storeMessages, nil
		}

		// No, fetch from the API.
		s.fewMutex.Unlock()
	} else {
		// Something wrong with the cached messages, make sure they aren't
		// returned.
		storeMessages = nil
	}

	// Store already has enough messages.
	if len(storeMessages) >= int(limit) && limit > 0 {
		return storeMessages[:limit], nil
	}

	// Decrease the limit, if we aren't fetching all messages.
	if limit > 0 {
		limit -= uint(len(storeMessages))
	}

	var before discord.MessageID = 0
	if len(storeMessages) > 0 {
		before = storeMessages[len(storeMessages)-1].ID
	}

	apiMessages, err := s.Session.MessagesBefore(channelID, before, limit)
	if err != nil {
		return nil, err
	}

	if len(storeMessages)+len(apiMessages) < s.MaxMessages() {
		// Tiny channel, store this.
		s.fewMutex.Lock()
		s.fewMessages[channelID] = struct{}{}
		s.fewMutex.Unlock()
	}

	if len(apiMessages) == 0 {
		return storeMessages, nil
	}

	// New messages fetched weirdly does not have GuildID filled. If we have
	// cached messages, we can use their GuildID. Otherwise, we need to fetch
	// it from the api.
	var guildID discord.GuildID
	if len(storeMessages) > 0 {
		guildID = storeMessages[0].GuildID
	} else {
		c, err := s.Channel(channelID)
		if err == nil {
			// If it's 0, it's 0 anyway. We don't need a check here.
			guildID = c.GuildID
		}
	}

	for i := range apiMessages {
		apiMessages[i].GuildID = guildID
	}

	if s.tracksMessage(&apiMessages[0]) && len(storeMessages) < s.MaxMessages() {
		// Only add as many messages as the store can hold.
		i := s.MaxMessages() - len(storeMessages)
		if i > len(apiMessages) {
			i = len(apiMessages)
		}

		msgs := apiMessages[:i]
		for i := range msgs {
			s.Cabinet.MessageSet(&msgs[i], false)
		}
	}

	return append(storeMessages, apiMessages...), nil
}

////

// Presence checks the state for user presences. If no guildID is given, it
// will look for the presence in all cached guilds.
func (s *State) Presence(gID discord.GuildID, uID discord.UserID) (*discord.Presence, error) {
	if !s.HasIntents(gateway.IntentGuildPresences) {
		return nil, store.ErrNotFound
	}

	// If there's no guild ID, look in all guilds
	if !gID.IsValid() {
		if !s.HasIntents(gateway.IntentGuilds) {
			return nil, store.ErrNotFound
		}

		g, err := s.Cabinet.Guilds()
		if err != nil {
			return nil, err
		}

		for _, g := range g {
			if p, err := s.Cabinet.Presence(g.ID, uID); err == nil {
				return p, nil
			}
		}

		return nil, store.ErrNotFound
	}

	return s.Cabinet.Presence(gID, uID)
}

////

func (s *State) Role(guildID discord.GuildID, roleID discord.RoleID) (target *discord.Role, err error) {
	if s.HasIntents(gateway.IntentGuilds) {
		target, err = s.Cabinet.Role(guildID, roleID)
		if err == nil {
			return
		}
	}

	rs, err := s.Session.Roles(guildID)
	if err != nil {
		return
	}

	for i, r := range rs {
		if r.ID == roleID {
			r := r // copy to prevent mem aliasing
			target = &r
		}

		if s.HasIntents(gateway.IntentGuilds) {
			s.RoleSet(guildID, &rs[i], false)
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

	if s.HasIntents(gateway.IntentGuilds) {
		for i := range rs {
			s.RoleSet(guildID, &rs[i], false)
		}
	}

	return rs, nil
}

func (s *State) fetchGuild(id discord.GuildID) (g *discord.Guild, err error) {
	g, err = s.Session.Guild(id)
	if err == nil && s.HasIntents(gateway.IntentGuilds) {
		s.Cabinet.GuildSet(g, false)
	}

	return
}

func (s *State) fetchMember(gID discord.GuildID, uID discord.UserID) (m *discord.Member, err error) {
	m, err = s.Session.Member(gID, uID)
	if err == nil && s.HasIntents(gateway.IntentGuildMembers) {
		s.Cabinet.MemberSet(gID, m, false)
	}

	return
}

// tracksMessage reports whether the state would track the passed message and
// messages from the same channel.
func (s *State) tracksMessage(m *discord.Message) bool {
	return (m.GuildID.IsValid() && s.HasIntents(gateway.IntentGuildMessages)) ||
		(!m.GuildID.IsValid() && s.HasIntents(gateway.IntentDirectMessages))
}

// tracksChannel reports whether the state would track the passed channel.
func (s *State) tracksChannel(c *discord.Channel) bool {
	return (c.GuildID.IsValid() && s.HasIntents(gateway.IntentGuilds)) ||
		!c.GuildID.IsValid()
}
