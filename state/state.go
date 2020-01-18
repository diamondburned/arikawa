package state

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/handler"
	"github.com/diamondburned/arikawa/session"
)

type State struct {
	*session.Session

	// PreHandler is the manual hook that is executed before the State handler
	// is. This should only be used for low-level operations.
	// It's recommended to set Synchronous to true if you mutate the events.
	PreHandler *handler.Handler // default nil

	MaxMessages uint // default 50

	Store

	unhooker func()
}

func NewFromSession(s *session.Session, store Store) (*State, error) {
	state := &State{
		Session: s,
		Store:   store,
	}

	return state, state.hookSession()
}

func New(token string) (*State, error) {
	return NewWithStore(token, NewDefaultStore(&DefaultStoreOptions{
		MaxMessages: 50,
	}))
}

func NewWithStore(token string, store Store) (*State, error) {
	s, err := session.New(token)
	if err != nil {
		return nil, err
	}

	state := &State{
		Session: s,
		Store:   store,
	}

	return state, state.hookSession()
}

// Unhook removes all state handlers from the session handlers.
func (s *State) Unhook() {
	s.unhooker()
}

////

func (s *State) Self() (*discord.User, error) {
	u, err := s.Store.Self()
	if err == nil {
		return u, nil
	}

	u, err = s.Session.Me()
	if err != nil {
		return nil, err
	}

	return u, s.Store.SelfSet(u)
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
		if err := s.Store.ChannelSet(&ch); err != nil {
			return nil, err
		}
	}

	return c, nil
}

////

func (s *State) hookSession() error {
	/*
		s.unhooker = s.Session.AddHandler(func(iface interface{}) {
			if s.PreHandler != nil {
				s.PreHandler.Call(iface)
			}

			s.mut.Lock()
			defer s.mut.Unlock()

			switch ev := iface.(type) {
			case *gateway.ReadyEvent:
				// Override
				s.guilds = ev.Guilds
				s.privates = ev.PrivateChannels
				s.self = ev.User

			case *gateway.MessageCreateEvent:
				_ = ev
				panic("IMPLEMENT ME")
			}
		})
	*/

	return nil
}
