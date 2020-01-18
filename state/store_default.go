package state

import (
	"sort"
	"sync"

	"github.com/diamondburned/arikawa/discord"
)

// TODO: make an ExpiryStore

type DefaultStore struct {
	*DefaultStoreOptions

	self discord.User

	// includes normal and private
	privates map[discord.Snowflake]*discord.Channel  // channelID:channel
	guilds   map[discord.Snowflake]*discord.Guild    // guildID:guild
	messages map[discord.Snowflake][]discord.Message // channelID:messages

	mut sync.Mutex
}

type DefaultStoreOptions struct {
	MaxMessages uint // default 50
}

var _ Store = (*DefaultStore)(nil)

func NewDefaultStore(opts *DefaultStoreOptions) *DefaultStore {
	if opts == nil {
		opts = &DefaultStoreOptions{
			MaxMessages: 50,
		}
	}

	return &DefaultStore{
		DefaultStoreOptions: opts,

		privates: map[discord.Snowflake]*discord.Channel{},
		guilds:   map[discord.Snowflake]*discord.Guild{},
		messages: map[discord.Snowflake][]discord.Message{},
	}
}

func (s *DefaultStore) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.self = discord.User{}

	s.privates = map[discord.Snowflake]*discord.Channel{}
	s.guilds = map[discord.Snowflake]*discord.Guild{}
	s.messages = map[discord.Snowflake][]discord.Message{}

	return nil
}

////

func (s *DefaultStore) Self() (*discord.User, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	if !s.self.ID.Valid() {
		return nil, ErrStoreNotFound
	}

	return &s.self, nil
}

func (s *DefaultStore) SetSelf(me *discord.User) error {
	s.mut.Lock()
	s.self = *me
	s.mut.Unlock()

	return nil
}

////

func (s *DefaultStore) Channel(id discord.Snowflake) (*discord.Channel, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	for _, g := range s.guilds {
		for _, ch := range g.Channels {
			if ch.ID == id {
				return &ch, nil
			}
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Channels(
	guildID discord.Snowflake) ([]discord.Channel, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return gd.Channels, nil
}

func (s *DefaultStore) PrivateChannels() ([]discord.Channel, error) {
	s.mut.Lock()

	var chs = make([]discord.Channel, 0, len(s.privates))
	for _, ch := range s.privates {
		chs = append(chs, *ch)
	}

	s.mut.Unlock()

	sort.Slice(chs, func(i, j int) bool {
		// Latest first
		return chs[i].LastMessageID > chs[j].LastMessageID
	})

	return chs, nil
}

func (s *DefaultStore) ChannelSet(channel *discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	switch channel.Type {
	case discord.DirectMessage, discord.GroupDM:
		s.privates[channel.ID] = channel

	default:
		gd, ok := s.guilds[channel.GuildID]
		if !ok {
			return ErrStoreNotFound
		}

		for i, ch := range gd.Channels {
			if ch.ID == channel.ID {
				// Found, just edit
				gd.Channels[i] = *channel
				return nil
			}
		}

		gd.Channels = append(gd.Channels, *channel)
	}

	return nil
}

func (s *DefaultStore) ChannelRemove(channel *discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[channel.GuildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, ch := range gd.Channels {
		if ch.ID == channel.ID {
			gd.Channels = append(gd.Channels[:i], gd.Channels[i+1:]...)
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Emoji(
	guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, emoji := range gd.Emojis {
		if emoji.ID == emojiID {
			return &emoji, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Emojis(
	guildID discord.Snowflake) ([]discord.Emoji, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return gd.Emojis, nil
}

func (s *DefaultStore) EmojiSet(
	guildID discord.Snowflake, emojis []discord.Emoji) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	filtered := emojis[:0]

Main:
	for _, enew := range emojis {
		// Try and see if this emoji is already in the slice
		for i, emoji := range gd.Emojis {
			if emoji.ID == enew.ID {
				// If it is, we simply replace it
				gd.Emojis[i] = enew

				continue Main
			}
		}

		// If not, we add it to the slice that's to be appended.
		filtered = append(filtered, enew)
	}

	// Append the new emojis
	gd.Emojis = append(gd.Emojis, filtered...)
	return nil
}

////

func (s *DefaultStore) Guild(id discord.Snowflake) (*discord.Guild, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	ch, ok := s.guilds[id]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return ch, nil
}

func (s *DefaultStore) Guilds() ([]discord.Guild, error) {
	s.mut.Lock()

	var gs = make([]discord.Guild, 0, len(s.guilds))
	for _, g := range s.guilds {
		gs = append(gs, *g)
	}

	s.mut.Unlock()

	sort.Slice(gs, func(i, j int) bool {
		return gs[i].ID > gs[j].ID
	})

	return gs, nil
}

func (s *DefaultStore) GuildSet(g *discord.Guild) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	old := s.guilds[g.ID]

	// Check the channels too
	if len(old.Channels) == 0 {
		return nil
	}

	if len(g.Channels) == 0 {
		g.Channels = old.Channels
	}

	s.guilds[g.ID] = g
	return nil
}

func (s *DefaultStore) GuildRemove(g *discord.Guild) error {
	s.mut.Lock()
	delete(s.guilds, g.ID)
	s.mut.Unlock()

	return nil
}

////

func (s *DefaultStore) Member(
	guildID, userID discord.Snowflake) (*discord.Member, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, member := range gd.Members {
		if member.User.ID == userID {
			return &member, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Members(
	guildID discord.Snowflake) ([]discord.Member, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return gd.Members, nil
}

func (s *DefaultStore) MemberSet(
	guildID discord.Snowflake, member *discord.Member) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	// Try and see if this member is already in the slice
	for i, m := range gd.Members {
		if m.User.ID == member.User.ID {
			// If it is, we simply replace it
			gd.Members[i] = *member
			return nil
		}
	}

	// Append the new member
	gd.Members = append(gd.Members, *member)
	return nil
}

func (s *DefaultStore) MemberRemove(guildID, userID discord.Snowflake) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	// Try and see if this member is already in the slice
	for i, m := range gd.Members {
		if m.User.ID == userID {
			gd.Members = append(gd.Members[:i], gd.Members[i+1:]...)
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Message(
	channelID, messageID discord.Snowflake) (*discord.Message, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[channelID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, m := range ms {
		if m.ID == messageID {
			return &m, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Messages(
	channelID discord.Snowflake) ([]discord.Message, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[channelID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return ms, nil
}

func (s *DefaultStore) MessageSet(message *discord.Message) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[message.ChannelID]
	if !ok {
		ms = make([]discord.Message, 0, int(s.MaxMessages)+1)
	}

	// Append
	ms = append(ms, *message)

	// Sort (should be fast since it's presorted)
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].ID > ms[j].ID
	})

	if len(ms) > int(s.MaxMessages) {
		ms = ms[len(ms)-int(s.MaxMessages):]
	}

	return nil
}

func (s *DefaultStore) MessageRemove(message *discord.Message) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[message.ChannelID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, m := range ms {
		if m.ID == message.ID {
			ms = append(ms[:i], ms[i+1:]...)
			s.messages[message.ChannelID] = ms
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Presence(
	guildID, userID discord.Snowflake) (*discord.Presence, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, p := range gd.Presences {
		if p.User.ID == userID {
			return &p, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Presences(
	guildID discord.Snowflake) ([]discord.Presence, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return gd.Presences, nil
}

func (s *DefaultStore) PresenceSet(
	guildID discord.Snowflake, presence *discord.Presence) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, p := range gd.Presences {
		if p.User.ID == presence.User.ID {
			gd.Presences[i] = *presence
			return nil
		}
	}

	gd.Presences = append(gd.Presences, *presence)
	return nil
}

func (s *DefaultStore) PresenceRemove(guildID, userID discord.Snowflake) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, p := range gd.Presences {
		if p.User.ID == userID {
			gd.Presences = append(gd.Presences[:i], gd.Presences[i+1:]...)
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Role(
	guildID, roleID discord.Snowflake) (*discord.Role, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, r := range gd.Roles {
		if r.ID == roleID {
			return &r, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Roles(
	guildID discord.Snowflake) ([]discord.Role, error) {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return gd.Roles, nil
}

func (s *DefaultStore) RoleSet(
	guildID discord.Snowflake, role *discord.Role) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, r := range gd.Roles {
		if r.ID == role.ID {
			gd.Roles[i] = *role
			return nil
		}
	}

	gd.Roles = append(gd.Roles, *role)
	return nil
}

func (s *DefaultStore) RoleRemove(guildID, roleID discord.Snowflake) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, r := range gd.Roles {
		if r.ID == roleID {
			gd.Roles = append(gd.Roles[:i], gd.Roles[i+1:]...)
			return nil
		}
	}

	return ErrStoreNotFound
}
