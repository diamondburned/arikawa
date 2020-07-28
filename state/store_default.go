package state

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
)

// TODO: make an ExpiryStore

type DefaultStore struct {
	DefaultStoreOptions

	self discord.User

	// includes normal and private
	privates map[discord.ChannelID]discord.Channel
	guilds   map[discord.GuildID]discord.Guild

	roles       map[discord.GuildID][]discord.Role
	emojis      map[discord.GuildID][]discord.Emoji
	channels    map[discord.GuildID][]discord.Channel
	presences   map[discord.GuildID][]discord.Presence
	voiceStates map[discord.GuildID][]discord.VoiceState
	messages    map[discord.ChannelID][]discord.Message

	// special case; optimize for lots of members
	members map[discord.GuildID]map[discord.UserID]discord.Member

	mut sync.RWMutex
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

	ds := &DefaultStore{DefaultStoreOptions: *opts}
	ds.Reset()

	return ds
}

func (s *DefaultStore) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.self = discord.User{}

	s.privates = map[discord.ChannelID]discord.Channel{}
	s.guilds = map[discord.GuildID]discord.Guild{}

	s.roles = map[discord.GuildID][]discord.Role{}
	s.emojis = map[discord.GuildID][]discord.Emoji{}
	s.channels = map[discord.GuildID][]discord.Channel{}
	s.presences = map[discord.GuildID][]discord.Presence{}
	s.voiceStates = map[discord.GuildID][]discord.VoiceState{}
	s.messages = map[discord.ChannelID][]discord.Message{}

	s.members = map[discord.GuildID]map[discord.UserID]discord.Member{}

	return nil
}

////

func (s *DefaultStore) Me() (*discord.User, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if !s.self.ID.Valid() {
		return nil, ErrStoreNotFound
	}

	return &s.self, nil
}

func (s *DefaultStore) MyselfSet(me discord.User) error {
	s.mut.Lock()
	s.self = me
	s.mut.Unlock()

	return nil
}

////

func (s *DefaultStore) Channel(id discord.ChannelID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if ch, ok := s.privates[id]; ok {
		// implicit copy
		return &ch, nil
	}

	for _, chs := range s.channels {
		for _, ch := range chs {
			if ch.ID == id {
				return &ch, nil
			}
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Channels(guildID discord.GuildID) ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	chs, ok := s.channels[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Channel{}, chs...), nil
}

// CreatePrivateChannel searches in the cache for a private channel. It makes no
// API calls.
func (s *DefaultStore) CreatePrivateChannel(recipient discord.UserID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	// slow way
	for _, ch := range s.privates {
		if ch.Type != discord.DirectMessage || len(ch.DMRecipients) == 0 {
			continue
		}
		if ch.DMRecipients[0].ID == recipient {
			// Return an implicit copy made by range.
			return &ch, nil
		}
	}
	return nil, ErrStoreNotFound
}

// PrivateChannels returns a list of Direct Message channels randomly ordered.
func (s *DefaultStore) PrivateChannels() ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	var chs = make([]discord.Channel, 0, len(s.privates))
	for i := range s.privates {
		chs = append(chs, s.privates[i])
	}

	return chs, nil
}

func (s *DefaultStore) ChannelSet(channel discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if !channel.GuildID.Valid() {
		s.privates[channel.ID] = channel

	} else {
		chs := s.channels[channel.GuildID]

		for i, ch := range chs {
			if ch.ID == channel.ID {
				// Also from discordgo.
				if channel.Permissions == nil {
					channel.Permissions = ch.Permissions
				}

				// Found, just edit
				chs[i] = channel

				return nil
			}
		}

		chs = append(chs, channel)
		s.channels[channel.GuildID] = chs
	}

	return nil
}

func (s *DefaultStore) ChannelRemove(channel discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	chs, ok := s.channels[channel.GuildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, ch := range chs {
		if ch.ID == channel.ID {
			// Fast unordered delete.
			chs[i] = chs[len(chs)-1]
			chs = chs[:len(chs)-1]

			s.channels[channel.GuildID] = chs
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Emoji(guildID discord.GuildID, emojiID discord.EmojiID) (*discord.Emoji, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	emojis, ok := s.emojis[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, emoji := range emojis {
		if emoji.ID == emojiID {
			// Emoji is an implicit copy, so we could do this safely.
			return &emoji, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Emojis(guildID discord.GuildID) ([]discord.Emoji, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	emojis, ok := s.emojis[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Emoji{}, emojis...), nil
}

func (s *DefaultStore) EmojiSet(guildID discord.GuildID, emojis []discord.Emoji) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// A nil slice is acceptable, as we'll make a new slice later on and set it.
	s.emojis[guildID] = emojis

	return nil
}

////

func (s *DefaultStore) Guild(id discord.GuildID) (*discord.Guild, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ch, ok := s.guilds[id]
	if !ok {
		return nil, ErrStoreNotFound
	}

	// implicit copy
	return &ch, nil
}

func (s *DefaultStore) Guilds() ([]discord.Guild, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if len(s.guilds) == 0 {
		return nil, ErrStoreNotFound
	}

	var gs = make([]discord.Guild, 0, len(s.guilds))
	for _, g := range s.guilds {
		gs = append(gs, g)
	}

	return gs, nil
}

func (s *DefaultStore) GuildSet(guild discord.Guild) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.guilds[guild.ID] = guild
	return nil
}

func (s *DefaultStore) GuildRemove(id discord.GuildID) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if _, ok := s.guilds[id]; !ok {
		return ErrStoreNotFound
	}

	delete(s.guilds, id)
	return nil
}

////

func (s *DefaultStore) Member(
	guildID discord.GuildID, userID discord.UserID) (*discord.Member, error) {

	s.mut.RLock()
	defer s.mut.RUnlock()

	ms, ok := s.members[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	m, ok := ms[userID]
	if ok {
		return &m, nil
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Members(guildID discord.GuildID) ([]discord.Member, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ms, ok := s.members[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	var members = make([]discord.Member, 0, len(ms))
	for _, m := range ms {
		members = append(members, m)
	}

	return members, nil
}

func (s *DefaultStore) MemberSet(guildID discord.GuildID, member discord.Member) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.members[guildID]
	if !ok {
		ms = make(map[discord.UserID]discord.Member, 1)
	}

	ms[member.User.ID] = member
	s.members[guildID] = ms

	return nil
}

func (s *DefaultStore) MemberRemove(guildID discord.GuildID, userID discord.UserID) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.members[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	if _, ok := ms[userID]; !ok {
		return ErrStoreNotFound
	}

	delete(ms, userID)
	return nil
}

////

func (s *DefaultStore) Message(
	channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {

	s.mut.RLock()
	defer s.mut.RUnlock()

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

func (s *DefaultStore) Messages(channelID discord.ChannelID) ([]discord.Message, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ms, ok := s.messages[channelID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Message{}, ms...), nil
}

func (s *DefaultStore) MaxMessages() int {
	return int(s.DefaultStoreOptions.MaxMessages)
}

func (s *DefaultStore) MessageSet(message discord.Message) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[message.ChannelID]
	if !ok {
		ms = make([]discord.Message, 0, s.MaxMessages()+1)
	}

	// Check if we already have the message.
	for i, m := range ms {
		if m.ID == message.ID {
			DiffMessage(message, &m)
			ms[i] = m
			return nil
		}
	}

	// Order: latest to earliest, similar to the API.

	var end = len(ms)
	if max := s.MaxMessages(); end >= max {
		// If the end (length) is larger than the maximum amount, then cap it.
		end = max
	} else {
		// Else, append an empty message to the end.
		ms = append(ms, discord.Message{})
		// Increment to update the length.
		end++
	}

	// Copy hack to prepend. This copies the 0th-(end-1)th entries to
	// 1st-endth.
	copy(ms[1:end], ms[0:end-1])
	// Then, set the 0th entry.
	ms[0] = message

	s.messages[message.ChannelID] = ms
	return nil
}

func (s *DefaultStore) MessageRemove(
	channelID discord.ChannelID, messageID discord.MessageID) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[channelID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, m := range ms {
		if m.ID == messageID {
			ms = append(ms[:i], ms[i+1:]...)
			s.messages[channelID] = ms
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Presence(
	guildID discord.GuildID, userID discord.UserID) (*discord.Presence, error) {

	s.mut.RLock()
	defer s.mut.RUnlock()

	ps, ok := s.presences[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, p := range ps {
		if p.User.ID == userID {
			return &p, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Presences(guildID discord.GuildID) ([]discord.Presence, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ps, ok := s.presences[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Presence{}, ps...), nil
}

func (s *DefaultStore) PresenceSet(guildID discord.GuildID, presence discord.Presence) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ps, _ := s.presences[guildID]

	for i, p := range ps {
		if p.User.ID == presence.User.ID {
			// Change the backing array.
			ps[i] = presence
			return nil
		}
	}

	ps = append(ps, presence)
	s.presences[guildID] = ps
	return nil
}

func (s *DefaultStore) PresenceRemove(guildID discord.GuildID, userID discord.UserID) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ps, ok := s.presences[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, p := range ps {
		if p.User.ID == userID {
			ps[i] = ps[len(ps)-1]
			ps = ps[:len(ps)-1]

			s.presences[guildID] = ps
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Role(guildID discord.GuildID, roleID discord.RoleID) (*discord.Role, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	rs, ok := s.roles[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, r := range rs {
		if r.ID == roleID {
			return &r, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Roles(guildID discord.GuildID) ([]discord.Role, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	rs, ok := s.roles[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Role{}, rs...), nil
}

func (s *DefaultStore) RoleSet(guildID discord.GuildID, role discord.Role) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// A nil slice is fine, since we can just append the role.
	rs, _ := s.roles[guildID]

	for i, r := range rs {
		if r.ID == role.ID {
			// This changes the backing array, so we don't need to reset the
			// slice.
			rs[i] = role
			return nil
		}
	}

	rs = append(rs, role)
	s.roles[guildID] = rs
	return nil
}

func (s *DefaultStore) RoleRemove(guildID discord.GuildID, roleID discord.RoleID) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	rs, ok := s.roles[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, r := range rs {
		if r.ID == roleID {
			// Fast delete.
			rs[i] = rs[len(rs)-1]
			rs = rs[:len(rs)-1]

			s.roles[guildID] = rs
			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) VoiceState(
	guildID discord.GuildID, userID discord.UserID) (*discord.VoiceState, error) {

	s.mut.RLock()
	defer s.mut.RUnlock()

	states, ok := s.voiceStates[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, vs := range states {
		if vs.UserID == userID {
			return &vs, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) VoiceStates(guildID discord.GuildID) ([]discord.VoiceState, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	states, ok := s.voiceStates[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.VoiceState{}, states...), nil
}

func (s *DefaultStore) VoiceStateSet(guildID discord.GuildID, voiceState discord.VoiceState) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	states, _ := s.voiceStates[guildID]

	for i, vs := range states {
		if vs.UserID == voiceState.UserID {
			// change the backing array
			states[i] = voiceState
			return nil
		}
	}

	states = append(states, voiceState)
	s.voiceStates[guildID] = states
	return nil
}

func (s *DefaultStore) VoiceStateRemove(guildID discord.GuildID, userID discord.UserID) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	states, ok := s.voiceStates[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, vs := range states {
		if vs.UserID == userID {
			states = append(states[:i], states[i+1:]...)
			s.voiceStates[guildID] = states

			return nil
		}
	}

	return ErrStoreNotFound
}
