// Package voice is coming soon to an arikawa near you!
package voice

import (
	"log"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/pkg/errors"
)

const (
	// Version represents the current version of the Discord Voice Gateway this package uses.
	Version = "4"

	// WSTimeout is the timeout for connecting and writing to the Websocket,
	// before Gateway cancels and fails.
	WSTimeout = wsutil.DefaultTimeout
)

var (
	// defaultErrorHandler is the default error handler
	defaultErrorHandler = func(err error) { log.Println("Voice gateway error:", err) }

	// WSDebug is used for extra debug logging. This is expected to behave
	// similarly to log.Println().
	WSDebug = func(v ...interface{}) {}

	// ErrMissingForIdentify .
	ErrMissingForIdentify = errors.New("missing GuildID, UserID, SessionID, or Token for identify")

	// ErrMissingForResume .
	ErrMissingForResume = errors.New("missing GuildID, SessionID, or Token for resuming")
)

// Voice .
type Voice struct {
	mut sync.RWMutex

	state *state.State

	// Connections holds all of the active voice connections.
	connections map[discord.Snowflake]*Connection

	// ErrorLog will be called when an error occurs (defaults to log.Println)
	ErrorLog func(err error)
}

// NewVoice creates a new Voice repository.
func NewVoice(s *state.State) *Voice {
	v := &Voice{
		state: s,

		connections: make(map[discord.Snowflake]*Connection),

		ErrorLog: defaultErrorHandler,
	}

	// Add the required event handlers to the session.
	s.AddHandler(v.onVoiceStateUpdate)
	s.AddHandler(v.onVoiceServerUpdate)

	return v
}

// GetConnection gets a connection for a guild with a read lock.
func (v *Voice) GetConnection(guildID discord.Snowflake) (*Connection, bool) {
	v.mut.RLock()
	defer v.mut.RUnlock()

	// For some reason you cannot just put `return v.connections[]` and return a bool D:
	conn, ok := v.connections[guildID]
	return conn, ok
}

// RemoveConnection removes a connection.
func (v *Voice) RemoveConnection(guildID discord.Snowflake) {
	v.mut.Lock()
	defer v.mut.Unlock()

	delete(v.connections, guildID)
}

// JoinChannel .
func (v *Voice) JoinChannel(gID, cID discord.Snowflake, muted, deafened bool) (*Connection, error) {
	// Get the stored voice connection for the given guild.
	conn, ok := v.GetConnection(gID)

	// Create a new voice connection if one does not exist.
	if !ok {
		conn = newConnection()

		v.mut.Lock()
		v.connections[gID] = conn
		v.mut.Unlock()
	}

	// Update values on the connection.
	conn.mut.Lock()
	conn.GuildID = gID
	conn.ChannelID = cID

	conn.muted = muted
	conn.deafened = deafened
	conn.mut.Unlock()

	// Ensure that if `cID` is zero that it passes null to the update event.
	var channelID *discord.Snowflake
	if cID != 0 {
		channelID = &cID
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	err := v.state.Gateway.UpdateVoiceState(gateway.UpdateVoiceStateData{
		GuildID:   gID,
		ChannelID: channelID,
		SelfMute:  muted,
		SelfDeaf:  deafened,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send Voice State Update event")
	}

	// Wait until we are connected.
	WSDebug("Waiting for READY.")

	// TODO: Find better way to wait for ready event.

	// Check if the connection is opened.
	for i := 0; i < 50; i++ {
		if conn.WS != nil && conn.WS.Conn != nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if conn.WS == nil || conn.WS.Conn == nil {
		return nil, errors.Wrap(err, "Failed to wait for connection to open")
	}

	for i := 0; i < 50; i++ {
		if conn.ready.IP != "" {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if conn.ready.IP == "" {
		return nil, errors.New("failed to wait for ready event")
	}

	if conn.udpConn == nil {
		return nil, errors.New("udp connection is not open")
	}

	WSDebug("Received READY.")

	return conn, nil
}
