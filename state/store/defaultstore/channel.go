package defaultstore

import (
	"errors"
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type Channel struct {
	mut sync.RWMutex

	// Channel references must be protected under the same mutex.

	privates map[discord.UserID]*discord.Channel
	channels map[discord.ChannelID]*discord.Channel
	guildChs map[discord.GuildID][]*discord.Channel
}

var _ store.ChannelStore = (*Channel)(nil)

func NewChannel() *Channel {
	return &Channel{
		privates: map[discord.UserID]*discord.Channel{},
		channels: map[discord.ChannelID]*discord.Channel{},
		guildChs: map[discord.GuildID][]*discord.Channel{},
	}
}

func (s *Channel) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.privates = map[discord.UserID]*discord.Channel{}
	s.channels = map[discord.ChannelID]*discord.Channel{}
	s.guildChs = map[discord.GuildID][]*discord.Channel{}

	return nil
}

func (s *Channel) Channel(id discord.ChannelID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	cpy := *ch
	return &cpy, nil
}

func (s *Channel) CreatePrivateChannel(recipient discord.UserID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ch, ok := s.privates[recipient]
	if !ok {
		return nil, store.ErrNotFound
	}

	cpy := *ch
	return &cpy, nil
}

// Channels returns a list of Guild channels randomly ordered.
func (s *Channel) Channels(guildID discord.GuildID) ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	chRefs, ok := s.guildChs[guildID]
	if !ok {
		return nil, store.ErrNotFound
	}

	// Reading chRefs is also covered by the global mutex.

	var channels = make([]discord.Channel, len(chRefs))
	for i, chRef := range chRefs {
		channels[i] = *chRef
	}

	return channels, nil
}

// PrivateChannels returns a list of Direct Message channels randomly ordered.
func (s *Channel) PrivateChannels() ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if len(s.privates) == 0 {
		return nil, store.ErrNotFound
	}

	var channels = make([]discord.Channel, 0, len(s.privates))
	for _, ch := range s.privates {
		channels = append(channels, *ch)
	}

	return channels, nil
}

// ChannelSet sets the Direct Message or Guild channl into the state. If the
// channel doesn't have 1 (one) DMRecipients, then it must have a valid GuildID,
// otherwise an error will be returned.
func (s *Channel) ChannelSet(channel discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// Update the reference if we can.
	if ch, ok := s.channels[channel.ID]; ok {
		*ch = channel
		return nil
	}

	if len(channel.DMRecipients) == 1 {
		s.privates[channel.DMRecipients[0].ID] = &channel
		s.channels[channel.ID] = &channel
		return nil
	}

	// Invalid channel case, as we need the GuildID to search for this channel.
	if !channel.GuildID.IsValid() {
		return errors.New("invalid guildID for guild channel")
	}

	// Always ensure that if the channel is in the slice, then it will be in the
	// map.

	s.channels[channel.ID] = &channel

	channels, _ := s.guildChs[channel.GuildID]
	channels = append(channels, &channel)
	s.guildChs[channel.GuildID] = channels

	return nil
}

func (s *Channel) ChannelRemove(channel discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	delete(s.channels, channel.ID)

	if len(channel.DMRecipients) == 1 {
		delete(s.privates, channel.DMRecipients[0].ID)
		return nil
	}

	channels, ok := s.guildChs[channel.GuildID]
	if !ok {
		return nil
	}

	for i, ch := range channels {
		if ch.ID != channel.ID {
			continue
		}

		// Fast unordered delete. Not sure if there's a benefit in doing
		// this over using a map, but I guess the memory usage is less and
		// there's no copying.

		// Move the last channel to the current channel, set the last
		// channel there to a nil value to unreference its children, then
		// slice the last channel off.
		channels[i] = channels[len(channels)-1]
		channels[len(channels)-1] = nil
		channels = channels[:len(channels)-1]

		s.guildChs[channel.GuildID] = channels

		break
	}

	return nil
}
