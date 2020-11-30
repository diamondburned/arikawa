package defaultstore

import (
	"sync"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type VoiceState struct {
	guilds moreatomic.Map
}

var _ store.VoiceStateStore = (*VoiceState)(nil)

type voiceStates struct {
	mut         sync.Mutex
	voiceStates map[discord.UserID]discord.VoiceState
}

func NewVoiceState() *VoiceState {
	return &VoiceState{
		guilds: *moreatomic.NewMap(func() interface{} {
			return &voiceStates{
				voiceStates: make(map[discord.UserID]discord.VoiceState, 1),
			}
		}),
	}
}

func (s *VoiceState) Reset() error {
	return s.guilds.Reset()
}

func (s *VoiceState) VoiceState(
	guildID discord.GuildID, userID discord.UserID) (*discord.VoiceState, error) {

	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	vs := iv.(*voiceStates)

	vs.mut.Lock()
	defer vs.mut.Unlock()

	v, ok := vs.voiceStates[userID]
	if ok {
		return &v, nil
	}

	return nil, store.ErrNotFound
}

func (s *VoiceState) VoiceStates(guildID discord.GuildID) ([]discord.VoiceState, error) {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil, store.ErrNotFound
	}

	vs := iv.(*voiceStates)

	vs.mut.Lock()
	defer vs.mut.Unlock()

	var states = make([]discord.VoiceState, 0, len(vs.voiceStates))
	for _, state := range vs.voiceStates {
		states = append(states, state)
	}

	return states, nil
}

func (s *VoiceState) VoiceStateSet(guildID discord.GuildID, voiceState discord.VoiceState) error {
	iv, _ := s.guilds.LoadOrStore(guildID)

	vs := iv.(*voiceStates)

	vs.mut.Lock()
	vs.voiceStates[voiceState.UserID] = voiceState
	vs.mut.Unlock()

	return nil
}

func (s *VoiceState) VoiceStateRemove(guildID discord.GuildID, userID discord.UserID) error {
	iv, ok := s.guilds.Load(guildID)
	if !ok {
		return nil
	}

	vs := iv.(*voiceStates)

	vs.mut.Lock()
	delete(vs.voiceStates, userID)
	vs.mut.Unlock()

	return nil
}
