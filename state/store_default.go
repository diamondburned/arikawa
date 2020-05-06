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
	privates map[discord.Snowflake]*discord.Channel // channelID:channel
	guilds   map[discord.Snowflake]*discord.Guild   // guildID:guild

	channels    map[discord.Snowflake][]discord.Channel    // guildID:channels
	members     map[discord.Snowflake][]discord.Member     // guildID:members
	presences   map[discord.Snowflake][]discord.Presence   // guildID:presences
	messages    map[discord.Snowflake][]discord.Message    // channelID:messages
	voiceStates map[discord.Snowflake][]discord.VoiceState // guildID:voiceStates

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

	ds := &DefaultStore{
		DefaultStoreOptions: opts,
	}
	ds.Reset()

	return ds
}

func (s *DefaultStore) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.self = discord.User{}

	s.privates = map[discord.Snowflake]*discord.Channel{}
	s.guilds = map[discord.Snowflake]*discord.Guild{}

	s.channels = map[discord.Snowflake][]discord.Channel{}
	s.members = map[discord.Snowflake][]discord.Member{}
	s.presences = map[discord.Snowflake][]discord.Presence{}
	s.messages = map[discord.Snowflake][]discord.Message{}
	s.voiceStates = map[discord.Snowflake][]discord.VoiceState{}

	return nil
}

////

func (s *DefaultStore) Me() (*discord.User, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	if !s.self.ID.Valid() {
		return nil, ErrStoreNotFound
	}

	return &s.self, nil
}

func (s *DefaultStore) MyselfSet(me *discord.User) error {
	s.mut.Lock()
	s.self = *me
	s.mut.Unlock()

	return nil
}

////

func (s *DefaultStore) Channel(id discord.Snowflake) (*discord.Channel, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	if ch, ok := s.privates[id]; ok {
		return ch, nil
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

func (s *DefaultStore) Channels(guildID discord.Snowflake) ([]discord.Channel, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	chs, ok := s.channels[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Channel{}, chs...), nil
}

// CreatePrivateChannel searches in the cache for a private channel. It makes no
// API calls.
func (s *DefaultStore) CreatePrivateChannel(recipient discord.Snowflake) (*discord.Channel, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	// slow way
	for _, ch := range s.privates {
		if ch.Type != discord.DirectMessage || len(ch.DMRecipients) < 1 {
			continue
		}
		if ch.DMRecipients[0].ID == recipient {
			return &(*ch), nil
		}
	}
	return nil, ErrStoreNotFound
}

// PrivateChannels returns a list of Direct Message channels randomly ordered.
func (s *DefaultStore) PrivateChannels() ([]discord.Channel, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	var chs = make([]discord.Channel, 0, len(s.privates))
	for _, ch := range s.privates {
		chs = append(chs, *ch)
	}

	return chs, nil
}

func (s *DefaultStore) ChannelSet(channel *discord.Channel) error {
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
				chs[i] = *channel

				return nil
			}
		}

		chs = append(chs, *channel)
		s.channels[channel.GuildID] = chs
	}

	return nil
}

func (s *DefaultStore) ChannelRemove(channel *discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	chs, ok := s.channels[channel.GuildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, ch := range chs {
		if ch.ID == channel.ID {
			chs = append(chs[:i], chs[i+1:]...)
			s.channels[channel.GuildID] = chs

			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Emoji(guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {
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

func (s *DefaultStore) Emojis(guildID discord.Snowflake) ([]discord.Emoji, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Emoji{}, gd.Emojis...), nil
}

func (s *DefaultStore) EmojiSet(guildID discord.Snowflake, emojis []discord.Emoji) error {
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

	if len(s.guilds) == 0 {
		s.mut.Unlock()
		return nil, ErrStoreNotFound
	}

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

func (s *DefaultStore) GuildSet(guild *discord.Guild) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if g, ok := s.guilds[guild.ID]; ok {
		// preserve state stuff
		if guild.Roles == nil {
			guild.Roles = g.Roles
		}
		if guild.Emojis == nil {
			guild.Emojis = g.Emojis
		}
	}

	s.guilds[guild.ID] = guild
	return nil
}

func (s *DefaultStore) GuildRemove(id discord.Snowflake) error {
	s.mut.Lock()
	delete(s.guilds, id)
	s.mut.Unlock()

	return nil
}

////

func (s *DefaultStore) Member(guildID, userID discord.Snowflake) (*discord.Member, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.members[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	for _, m := range ms {
		if m.User.ID == userID {
			return &m, nil
		}
	}

	return nil, ErrStoreNotFound
}

func (s *DefaultStore) Members(guildID discord.Snowflake) ([]discord.Member, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.members[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Member{}, ms...), nil
}

func (s *DefaultStore) MemberSet(guildID discord.Snowflake, member *discord.Member) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms := s.members[guildID]

	// Try and see if this member is already in the slice
	for i, m := range ms {
		if m.User.ID == member.User.ID {
			// If it is, we simply replace it
			ms[i] = *member
			s.members[guildID] = ms

			return nil
		}
	}

	// Append the new member
	ms = append(ms, *member)
	s.members[guildID] = ms

	return nil
}

func (s *DefaultStore) MemberRemove(guildID, userID discord.Snowflake) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.members[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	// Try and see if this member is already in the slice
	for i, m := range ms {
		if m.User.ID == userID {
			ms = append(ms, ms[i+1:]...)
			s.members[guildID] = ms

			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Message(channelID, messageID discord.Snowflake) (*discord.Message, error) {
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

func (s *DefaultStore) Messages(channelID discord.Snowflake) ([]discord.Message, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[channelID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return ms, nil
}

func (s *DefaultStore) MaxMessages() int {
	return int(s.DefaultStoreOptions.MaxMessages)
}

func (s *DefaultStore) MessageSet(message *discord.Message) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ms, ok := s.messages[message.ChannelID]
	if !ok {
		ms = make([]discord.Message, 0, s.MaxMessages()+1)
	}

	// Check if we already have the message.
	for i, m := range ms {
		if m.ID == message.ID {
			DiffMessage(*message, &m)
			ms[i] = m
			return nil
		}
	}

	// Prepend the latest message at the end

	if end := s.MaxMessages(); len(ms) >= end {
		// Copy hack to prepend. This copies the 0th-(end-1)th entries to
		// 1st-endth.
		copy(ms[1:end], ms[0:end-1])
		// Then, set the 0th entry.
		ms[0] = *message

	} else {
		ms = append(ms, *message)
	}

	s.messages[message.ChannelID] = ms
	return nil
}

func (s *DefaultStore) MessageRemove(channelID, messageID discord.Snowflake) error {
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

func (s *DefaultStore) Presence(guildID, userID discord.Snowflake) (*discord.Presence, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

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

func (s *DefaultStore) Presences(guildID discord.Snowflake) ([]discord.Presence, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	ps, ok := s.presences[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return ps, nil
}

func (s *DefaultStore) PresenceSet(guildID discord.Snowflake, presence *discord.Presence) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ps := s.presences[guildID]

	for i, p := range ps {
		if p.User.ID == presence.User.ID {
			ps[i] = *presence
			s.presences[guildID] = ps

			return nil
		}
	}

	ps = append(ps, *presence)
	s.presences[guildID] = ps
	return nil
}

func (s *DefaultStore) PresenceRemove(guildID, userID discord.Snowflake) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	ps, ok := s.presences[guildID]
	if !ok {
		return ErrStoreNotFound
	}

	for i, p := range ps {
		if p.User.ID == userID {
			ps = append(ps[:i], ps[i+1:]...)
			s.presences[guildID] = ps

			return nil
		}
	}

	return ErrStoreNotFound
}

////

func (s *DefaultStore) Role(guildID, roleID discord.Snowflake) (*discord.Role, error) {
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

func (s *DefaultStore) Roles(guildID discord.Snowflake) ([]discord.Role, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	gd, ok := s.guilds[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.Role{}, gd.Roles...), nil
}

func (s *DefaultStore) RoleSet(guildID discord.Snowflake, role *discord.Role) error {
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

////

func (s *DefaultStore) VoiceState(guildID, userID discord.Snowflake) (*discord.VoiceState, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

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

func (s *DefaultStore) VoiceStates(guildID discord.Snowflake) ([]discord.VoiceState, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	states, ok := s.voiceStates[guildID]
	if !ok {
		return nil, ErrStoreNotFound
	}

	return append([]discord.VoiceState{}, states...), nil
}

func (s *DefaultStore) VoiceStateSet(guildID discord.Snowflake, voiceState *discord.VoiceState) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	states := s.voiceStates[guildID]

	for i, vs := range states {
		if vs.UserID == voiceState.UserID {
			states[i] = *voiceState
			s.voiceStates[guildID] = states

			return nil
		}
	}

	states = append(states, *voiceState)
	s.voiceStates[guildID] = states
	return nil
}

func (s *DefaultStore) VoiceStateRemove(guildID, userID discord.Snowflake) error {
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
