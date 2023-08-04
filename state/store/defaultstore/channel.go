package defaultstore

import (
	"errors"
	"fmt"
	"sync"

	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/state/store"
)

type Channel struct {
	mut sync.RWMutex

	// Channel references must be protected under the same mutex.

	channels map[discord.ChannelID]discord.Channel
	privates map[discord.UserID]discord.ChannelID
	guildChs map[discord.GuildID][]discord.ChannelID
}

var _ store.ChannelStore = (*Channel)(nil)

func NewChannel() *Channel {
	return &Channel{
		channels: map[discord.ChannelID]discord.Channel{},
		privates: map[discord.UserID]discord.ChannelID{},
		guildChs: map[discord.GuildID][]discord.ChannelID{},
	}
}

func (s *Channel) Reset() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.channels = map[discord.ChannelID]discord.Channel{}
	s.privates = map[discord.UserID]discord.ChannelID{}
	s.guildChs = map[discord.GuildID][]discord.ChannelID{}

	return nil
}

func (s *Channel) Channel(id discord.ChannelID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	return &ch, nil
}

func (s *Channel) CreatePrivateChannel(recipient discord.UserID) (*discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	id, ok := s.privates[recipient]
	if !ok {
		return nil, store.ErrNotFound
	}

	cpy := s.channels[id]
	return &cpy, nil
}

// Channels returns a list of Guild channels randomly ordered.
func (s *Channel) Channels(guildID discord.GuildID) ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	chIDs, ok := s.guildChs[guildID]
	if !ok {
		return nil, store.ErrNotFound
	}

	// Reading chRefs is also covered by the global mutex.

	var channels = make([]discord.Channel, 0, len(chIDs))
	for _, chID := range chIDs {
		ch, ok := s.channels[chID]
		if !ok {
			continue
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

// PrivateChannels returns a list of Direct Message channels randomly ordered.
func (s *Channel) PrivateChannels() ([]discord.Channel, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	groupDMs := s.guildChs[0]

	if len(s.privates) == 0 && len(groupDMs) == 0 {
		return nil, store.ErrNotFound
	}

	var channels = make([]discord.Channel, 0, len(s.privates)+len(groupDMs))
	for _, chID := range s.privates {
		if ch, ok := s.channels[chID]; ok {
			channels = append(channels, ch)
		}
	}
	for _, chID := range groupDMs {
		if ch, ok := s.channels[chID]; ok {
			channels = append(channels, ch)
		}
	}

	return channels, nil
}

// ChannelSet sets the Direct Message or Guild channel into the state.
func (s *Channel) ChannelSet(channel *discord.Channel, update bool) error {
	cpy := *channel

	s.mut.Lock()
	defer s.mut.Unlock()

	// Update the reference if we can.
	s.channels[channel.ID] = cpy

	switch channel.Type {
	case discord.DirectMessage:
		// Safety bound check.
		if len(channel.DMRecipients) != 1 {
			return fmt.Errorf("DirectMessage channel %d doesn't have 1 recipient", channel.ID)
		}
		s.privates[channel.DMRecipients[0].ID] = channel.ID
		return nil
	case discord.GroupDM:
		s.guildChs[0] = addChannelID(s.guildChs[0], channel.ID)
		return nil
	}

	// Ensure that if the channel is not a DM or group DM channel, then it must
	// have a valid guild ID.
	if !channel.GuildID.IsValid() {
		return errors.New("invalid guildID for guild channel")
	}

	s.guildChs[channel.GuildID] = addChannelID(s.guildChs[channel.GuildID], channel.ID)
	return nil
}

func (s *Channel) ChannelRemove(channel *discord.Channel) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// Wipe the channel off the channel ID index.
	delete(s.channels, channel.ID)

	// Wipe the channel off the DM recipient index, if available.
	switch channel.Type {
	case discord.DirectMessage:
		// Safety bound check.
		if len(channel.DMRecipients) != 1 {
			return fmt.Errorf("DirectMessage channel %d doesn't have 1 recipient", channel.ID)
		}
		delete(s.privates, channel.DMRecipients[0].ID)
		return nil
	case discord.GroupDM:
		s.guildChs[0] = removeChannelID(s.guildChs[0], channel.ID)
		return nil
	}

	s.guildChs[channel.GuildID] = removeChannelID(s.guildChs[channel.GuildID], channel.ID)
	return nil
}

func addChannelID(channels []discord.ChannelID, id discord.ChannelID) []discord.ChannelID {
	for _, ch := range channels {
		if ch == id {
			return channels
		}
	}
	if channels == nil {
		channels = make([]discord.ChannelID, 0, 5)
	}
	return append(channels, id)
}

// removeChannelID removes the given channel with the index from the given
// channels slice in an unordered fashion.
func removeChannelID(channels []discord.ChannelID, id discord.ChannelID) []discord.ChannelID {
	for i, ch := range channels {
		if ch == id {
			// Move the last channel to the current channel, then slice the last
			// channel off.
			channels[i] = channels[len(channels)-1]
			channels = channels[:len(channels)-1]
			break
		}
	}
	return channels
}
