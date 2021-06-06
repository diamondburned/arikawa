package defaultstore

import (
	"errors"
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state/store"
)

type Channel struct {
	privates map[discord.UserID]*discord.Channel
	// channels references must be protected under the same mutex.
	channels   map[discord.ChannelID]*discord.Channel
	guildChs   map[discord.GuildID][]*discord.Channel
	privateChs []*discord.Channel
	mut        sync.RWMutex
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

	channels := make([]discord.Channel, len(chRefs))
	for i, chRef := range chRefs {
		channels[i] = *chRef
	}

	return channels, nil
}

// PrivateChannels returns a list of Direct Message channels randomly ordered.
func (s *Channel) PrivateChannels() ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if len(s.privateChs) == 0 {
		return nil, store.ErrNotFound
	}

	channels := make([]discord.Channel, len(s.privateChs))
	for i, ch := range s.privateChs {
		channels[i] = *ch
	}

	return channels, nil
}

// ChannelSet sets the Direct Message or Guild channel into the state.
func (s *Channel) ChannelSet(channel discord.Channel, update bool) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// Update the reference if we can.
	if ch, ok := s.channels[channel.ID]; ok {
		if update {
			*ch = channel
		}
		return nil
	}

	switch channel.Type {
	case discord.DirectMessage:
		// Safety bound check.
		if len(channel.DMRecipients) != 1 {
			return errors.New("DirectMessage channel does not have 1 recipient")
		}
		s.privates[channel.DMRecipients[0].ID] = &channel
		fallthrough
	case discord.GroupDM:
		s.privateChs = append(s.privateChs, &channel)
		s.channels[channel.ID] = &channel
		return nil
	}

	// Ensure that if the channel is not a DM or group DM channel, then it must
	// have a valid guild ID.
	if !channel.GuildID.IsValid() {
		return errors.New("invalid guildID for guild channel")
	}

	s.channels[channel.ID] = &channel

	channels := s.guildChs[channel.GuildID]
	channels = append(channels, &channel)
	s.guildChs[channel.GuildID] = channels

	return nil
}

func (s *Channel) ChannelRemove(channel discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// Wipe the channel off the channel ID index.
	delete(s.channels, channel.ID)

	// Wipe the channel off the DM recipient index, if available.
	switch channel.Type {
	case discord.DirectMessage:
		// Safety bound check.
		if len(channel.DMRecipients) != 1 {
			return errors.New("DirectMessage channel does not have 1 recipient")
		}
		delete(s.privates, channel.DMRecipients[0].ID)
		fallthrough
	case discord.GroupDM:
		for i, priv := range s.privateChs {
			if priv.ID == channel.ID {
				s.privateChs = removeChannel(s.privateChs, i)
				break
			}
		}
		return nil
	}

	// Wipe the channel off the guilds index, if available.
	channels, ok := s.guildChs[channel.GuildID]
	if !ok {
		return nil
	}

	for i, ch := range channels {
		if ch.ID == channel.ID {
			s.guildChs[channel.GuildID] = removeChannel(channels, i)
			break
		}
	}

	return nil
}

// removeChannel removes the given channel with the index from the given
// channels slice in an unordered fashion.
func removeChannel(channels []*discord.Channel, i int) []*discord.Channel {
	// Fast unordered delete. Not sure if there's a benefit in doing
	// this over using a map, but I guess the memory usage is less and
	// there's no copying.

	// Move the last channel to the current channel, set the last
	// channel there to a nil value to unreference its children, then
	// slice the last channel off.
	channels[i] = channels[len(channels)-1]
	channels[len(channels)-1] = nil
	channels = channels[:len(channels)-1]

	return channels
}
