// Package voice handles the Discord voice gateway and UDP connections, as well
// as managing and keeping track of multiple voice sessions.
//
// This package abstracts the subpackage voice/voicesession and voice/udp.
package voice

import (
	"log"
	"strconv"
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/pkg/errors"
)

var (
	// defaultErrorHandler is the default error handler
	defaultErrorHandler = func(err error) { log.Println("voice gateway error:", err) }

	// ErrCannotSend is an error when audio is sent to a closed channel.
	ErrCannotSend = errors.New("cannot send audio to closed channel")
)

// Voice represents a Voice Repository used for managing voice sessions.
type Voice struct {
	*state.State

	// Session holds all of the active voice sessions.
	mapmutex sync.Mutex
	sessions map[discord.GuildID]*Session

	// Callbacks to remove the handlers.
	closers []func()

	// ErrorLog will be called when an error occurs (defaults to log.Println)
	ErrorLog func(err error)
}

// NewVoiceFromToken creates a new voice session from the given token.
func NewVoiceFromToken(token string) (*Voice, error) {
	s, err := state.New(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new session")
	}

	return NewVoice(s), nil
}

// NewVoice creates a new Voice repository wrapped around a state. The function
// will also automatically add the GuildVoiceStates intent, as that is required.
func NewVoice(s *state.State) *Voice {
	v := &Voice{
		State:    s,
		sessions: make(map[discord.GuildID]*Session),
		ErrorLog: defaultErrorHandler,
	}

	// Add the required event handlers to the session.
	v.closers = []func(){
		s.AddHandler(v.onVoiceStateUpdate),
		s.AddHandler(v.onVoiceServerUpdate),
	}

	return v
}

// onVoiceStateUpdate receives VoiceStateUpdateEvents from the gateway
// to keep track of the current user's voice state.
func (v *Voice) onVoiceStateUpdate(e *gateway.VoiceStateUpdateEvent) {
	// Get the current user.
	me, err := v.Me()
	if err != nil {
		v.ErrorLog(err)
		return
	}

	// Ignore the event if it is an update from another user.
	if me.ID != e.UserID {
		return
	}

	// Get the stored voice session for the given guild.
	vs, ok := v.GetSession(e.GuildID)
	if !ok {
		return
	}

	// Do what we must.
	vs.UpdateState(e)

	// Remove the connection if the current user has disconnected.
	if e.ChannelID == 0 {
		v.RemoveSession(e.GuildID)
	}
}

// onVoiceServerUpdate receives VoiceServerUpdateEvents from the gateway
// to manage the current user's voice connections.
func (v *Voice) onVoiceServerUpdate(e *gateway.VoiceServerUpdateEvent) {
	// Get the stored voice session for the given guild.
	vs, ok := v.GetSession(e.GuildID)
	if !ok {
		return
	}

	// Do what we must.
	vs.UpdateServer(e)
}

// GetSession gets a session for a guild with a read lock.
func (v *Voice) GetSession(guildID discord.GuildID) (*Session, bool) {
	v.mapmutex.Lock()
	defer v.mapmutex.Unlock()

	// For some reason you cannot just put `return v.sessions[]` and return a bool D:
	conn, ok := v.sessions[guildID]
	return conn, ok
}

// RemoveSession removes a session.
func (v *Voice) RemoveSession(guildID discord.GuildID) {
	v.mapmutex.Lock()
	defer v.mapmutex.Unlock()

	// Ensure that the session is disconnected.
	if ses, ok := v.sessions[guildID]; ok {
		ses.Disconnect()
	}

	delete(v.sessions, guildID)
}

// JoinChannel joins the specified channel in the specified guild.
func (v *Voice) JoinChannel(gID discord.GuildID, cID discord.ChannelID, muted, deafened bool) (*Session, error) {
	// Get the stored voice session for the given guild.
	conn, ok := v.GetSession(gID)

	// Create a new voice session if one does not exist.
	if !ok {
		u, err := v.Me()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get self")
		}

		conn = NewSession(v.Session, u.ID)
		conn.ErrorLog = v.ErrorLog

		v.mapmutex.Lock()
		v.sessions[gID] = conn
		v.mapmutex.Unlock()
	}

	// Connect.
	return conn, conn.JoinChannel(gID, cID, muted, deafened)
}

func (v *Voice) Close() error {
	err := &CloseError{
		SessionErrors: make(map[discord.GuildID]error),
	}

	v.mapmutex.Lock()
	defer v.mapmutex.Unlock()

	// Remove all callback handlers.
	for _, fn := range v.closers {
		fn()
	}

	for gID, s := range v.sessions {
		if dErr := s.Disconnect(); dErr != nil {
			err.SessionErrors[gID] = dErr
		}
	}

	err.StateErr = v.State.Close()
	if err.HasError() {
		return err
	}

	return nil
}

type CloseError struct {
	SessionErrors map[discord.GuildID]error
	StateErr      error
}

func (e *CloseError) HasError() bool {
	if e.StateErr != nil {
		return true
	}

	return len(e.SessionErrors) > 0
}

func (e *CloseError) Error() string {
	if e.StateErr != nil {
		return e.StateErr.Error()
	}

	if len(e.SessionErrors) < 1 {
		return ""
	}

	return strconv.Itoa(len(e.SessionErrors)) + " voice sessions returned errors while attempting to disconnect"
}
