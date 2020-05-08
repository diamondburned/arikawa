package voice

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
	"github.com/diamondburned/arikawa/utils/moreatomic"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/diamondburned/arikawa/voice/udp"
	"github.com/diamondburned/arikawa/voice/voicegateway"
	"github.com/pkg/errors"
)

const Protocol = "xsalsa20_poly1305"

var OpusSilence = [...]byte{0xF8, 0xFF, 0xFE}

type Session struct {
	session *session.Session
	state   voicegateway.State

	ErrorLog func(err error)

	// Filled by events.
	// sessionID string
	// token     string
	// endpoint  string

	// joining determines the behavior of incoming event callbacks (Update).
	// If this is true, incoming events will just send into Updated channels. If
	// false, events will trigger a reconnection.
	joining  moreatomic.Bool
	incoming chan struct{} // used only when joining == true

	mut sync.RWMutex

	// TODO: expose getters mutex-guarded.
	gateway  *voicegateway.Gateway
	voiceUDP *udp.Connection

	muted    bool
	deafened bool
	speaking bool
}

func NewSession(ses *session.Session, userID discord.Snowflake) *Session {
	return &Session{
		session: ses,
		state: voicegateway.State{
			UserID: userID,
		},
		ErrorLog: func(err error) {},
		incoming: make(chan struct{}),
	}
}

func (s *Session) UpdateServer(ev *gateway.VoiceServerUpdateEvent) {
	// If this is true, then mutex is acquired already.
	if s.joining.Get() {
		s.state.Endpoint = ev.Endpoint
		s.state.Token = ev.Token

		s.incoming <- struct{}{}
		return
	}

	// Reconnect.
	s.mut.Lock()
	defer s.mut.Unlock()

	s.state.Endpoint = ev.Endpoint
	s.state.Token = ev.Token

	if err := s.reconnect(); err != nil {
		s.ErrorLog(errors.Wrap(err, "Failed to reconnect after voice server update"))
	}
}

func (s *Session) UpdateState(ev *gateway.VoiceStateUpdateEvent) {
	if s.state.UserID != ev.UserID {
		// Not our state.
		return
	}

	// If this is true, then mutex is acquired already.
	if s.joining.Get() {
		s.state.SessionID = ev.SessionID
		s.state.ChannelID = ev.ChannelID

		s.incoming <- struct{}{}
		return
	}
}

func (s *Session) JoinChannel(gID, cID discord.Snowflake, muted, deafened bool) error {
	// Acquire the mutex during join, locking during IO as well.
	s.mut.Lock()
	defer s.mut.Unlock()

	// Set that we're joining.
	s.joining.Set(true)
	defer s.joining.Set(false) // reset when done

	// ensure gateeway and voiceUDP is already closed.
	s.ensureClosed()

	// Set the state.
	s.state.ChannelID = cID
	s.state.GuildID = gID

	s.muted = muted
	s.deafened = deafened
	s.speaking = false

	// Ensure that if `cID` is zero that it passes null to the update event.
	var channelID discord.Snowflake = -1
	if cID.Valid() {
		channelID = cID
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	err := s.session.Gateway.UpdateVoiceState(gateway.UpdateVoiceStateData{
		GuildID:   gID,
		ChannelID: channelID,
		SelfMute:  muted,
		SelfDeaf:  deafened,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to send Voice State Update event")
	}

	// Wait for replies. The above command should reply with these 2 events.
	<-s.incoming
	<-s.incoming

	// These 2 methods should've updated s.state before sending into these
	// channels. Since s.state is already filled, we can go ahead and connect.

	return s.reconnect()
}

// reconnect uses the current state to reconnect to a new gateway and UDP
// connection.
func (s *Session) reconnect() (err error) {
	s.gateway = voicegateway.New(s.state)

	// Open the voice gateway. The function will block until Ready is received.
	if err := s.gateway.Open(); err != nil {
		return errors.Wrap(err, "Failed to open voice gateway")
	}

	// Get the Ready event.
	voiceReady := s.gateway.Ready()

	// Prepare the UDP voice connection.
	s.voiceUDP, err = udp.DialConnection(voiceReady.Addr(), voiceReady.SSRC)
	if err != nil {
		return errors.Wrap(err, "Failed to open voice UDP connection")
	}

	// Get the session description from the voice gateway.
	d, err := s.gateway.SessionDescription(voicegateway.SelectProtocol{
		Protocol: "udp",
		Data: voicegateway.SelectProtocolData{
			Address: s.voiceUDP.GatewayIP,
			Port:    s.voiceUDP.GatewayPort,
			Mode:    Protocol,
		},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to select protocol")
	}

	// Start the UDP loop.
	go s.voiceUDP.Start(&d.SecretKey)

	return nil
}

// Speaking tells Discord we're speaking. This calls
// (*voicegateway.Gateway).Speaking().
func (s *Session) Speaking(flag voicegateway.SpeakingFlag) error {
	// TODO: maybe we don't need to mutex protect IO.
	s.mut.RLock()
	defer s.mut.RUnlock()

	return s.gateway.Speaking(flag)
}

func (s *Session) StopSpeaking() error {
	// Send 5 frames of silence.
	for i := 0; i < 5; i++ {
		if _, err := s.Write(OpusSilence[:]); err != nil {
			return errors.Wrapf(err, "Failed to send frame %d", i)
		}
	}
	return nil
}

func (s *Session) Write(b []byte) (int, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if s.voiceUDP == nil {
		return 0, ErrCannotSend
	}
	return s.voiceUDP.Write(b)
}

func (s *Session) Disconnect() error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// If we're already closed.
	if s.gateway == nil && s.voiceUDP == nil {
		return nil
	}

	// Notify Discord that we're leaving. This will send a
	// VoiceStateUpdateEvent, in which our handler will promptly remove the
	// session from the map.

	err := s.session.Gateway.UpdateVoiceState(gateway.UpdateVoiceStateData{
		GuildID:   s.state.GuildID,
		ChannelID: discord.NullSnowflake,
		SelfMute:  true,
		SelfDeaf:  true,
	})

	s.ensureClosed()
	// wrap returns nil if err is nil
	return errors.Wrap(err, "Failed to update voice state")
}

// close ensures everything is closed. It does not acquire the mutex.
func (s *Session) ensureClosed() {
	// If we're already closed.
	if s.gateway == nil && s.voiceUDP == nil {
		return
	}

	// Disconnect the UDP connection.
	if s.voiceUDP != nil {
		s.voiceUDP.Close()
		s.voiceUDP = nil
	}

	// Disconnect the voice gateway, ignoring the error.
	if s.gateway != nil {
		if err := s.gateway.Close(); err != nil {
			wsutil.WSDebug("Uncaught voice gateway close error:", err)
		}
		s.gateway = nil
	}
}
