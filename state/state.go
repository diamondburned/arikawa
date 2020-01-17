package state

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/handler"
	"github.com/diamondburned/arikawa/session"
)

type State struct {
	*session.Session

	// PreHandler is the manual hook that is executed before the State handler
	// is. This should only be used for low-level operations.
	// It's recommended to set Synchronous to true if you mutate the events.
	PreHandler *handler.Handler

	guilds   []discord.Guild
	channels []discord.Channel
	privates []discord.Channel
	messages map[discord.Snowflake][]discord.Message

	mut sync.Mutex

	unhooker func()
}

func NewFromSession(s *session.Session) (*State, error) {
	state := &State{
		Session:  s,
		messages: map[discord.Snowflake][]discord.Message{},
	}

	return state, state.hookSession()
}

// Unhook removes all state handlers from the session handlers.
func (s *State) Unhook() {
	s.unhooker()
}

// Reset resets the entire state.
func (s *State) Reset() {
	s.mut.Lock()
	defer s.mut.Unlock()

	panic("IMPLEMENT ME")
}

func (s *State) hookSession() error {
	s.unhooker = s.Session.AddHandler(func(iface interface{}) {
		if s.PreHandler != nil {
			s.PreHandler.Call(iface)
		}

		switch ev := iface.(type) {
		case *gateway.ReadyEvent:
		case *gateway.MessageCreateEvent:
			_ = ev
			panic("IMPLEMENT ME")
		}
	})
	return nil
}
