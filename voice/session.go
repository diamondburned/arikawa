package voice

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/internal/moreatomic"
	"github.com/diamondburned/arikawa/session"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/diamondburned/arikawa/voice/udp"
	"github.com/diamondburned/arikawa/voice/voicegateway"
)

const Protocol = "xsalsa20_poly1305"

var OpusSilence = [...]byte{0xF8, 0xFF, 0xFE}

// WSTimeout is the duration to wait for a gateway operation including Session
// to complete before erroring out. This only applies to functions that don't
// take in a context already.
var WSTimeout = 10 * time.Second

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

func NewSession(ses *session.Session, userID discord.UserID) *Session {
	return &Session{
		session: ses,
		state: voicegateway.State{
			UserID: userID,
		},
		ErrorLog: func(err error) {},
		incoming: make(chan struct{}, 2),
	}
}

func (s *Session) UpdateServer(ev *gateway.VoiceServerUpdateEvent) {
	if s.state.GuildID != ev.GuildID {
		// Not our state.
		return
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	if err := s.reconnectCtx(ctx); err != nil {
		s.ErrorLog(errors.Wrap(err, "failed to reconnect after voice server update"))
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

func (s *Session) JoinChannel(gID discord.GuildID, cID discord.ChannelID, muted, deafened bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.JoinChannelCtx(ctx, gID, cID, muted, deafened)
}

func (s *Session) JoinChannelCtx(ctx context.Context, gID discord.GuildID, cID discord.ChannelID, muted, deafened bool) error {
	// Acquire the mutex during join, locking during IO as well.
	s.mut.Lock()
	defer s.mut.Unlock()

	// Set that we're joining.
	s.joining.Set(true)
	defer s.joining.Set(false) // reset when done

	// Ensure gateway and voiceUDP are already closed.
	s.ensureClosed()

	// Set the state.
	s.state.ChannelID = cID
	s.state.GuildID = gID

	s.muted = muted
	s.deafened = deafened
	s.speaking = false

	// Ensure that if `cID` is zero that it passes null to the update event.
	var channelID discord.ChannelID = -1
	if cID.Valid() {
		channelID = cID
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	err := s.session.Gateway.UpdateVoiceStateCtx(ctx, gateway.UpdateVoiceStateData{
		GuildID:   gID,
		ChannelID: channelID,
		SelfMute:  muted,
		SelfDeaf:  deafened,
	})
	if err != nil {
		return errors.Wrap(err, "failed to send Voice State Update event")
	}

	// Wait for 2 replies. The above command should reply with these 2 events.
	if err := s.waitForIncoming(ctx, 2); err != nil {
		return errors.Wrap(err, "failed to wait for needed gateway events")
	}

	// These 2 methods should've updated s.state before sending into these
	// channels. Since s.state is already filled, we can go ahead and connect.

	return s.reconnectCtx(ctx)
}

func (s *Session) waitForIncoming(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		select {
		case <-s.incoming:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// reconnect uses the current state to reconnect to a new gateway and UDP
// connection.
func (s *Session) reconnectCtx(ctx context.Context) (err error) {
	s.gateway = voicegateway.New(s.state)

	// Open the voice gateway. The function will block until Ready is received.
	if err := s.gateway.OpenCtx(ctx); err != nil {
		return errors.Wrap(err, "failed to open voice gateway")
	}

	// Get the Ready event.
	voiceReady := s.gateway.Ready()

	// Prepare the UDP voice connection.
	s.voiceUDP, err = udp.DialConnectionCtx(ctx, voiceReady.Addr(), voiceReady.SSRC)
	if err != nil {
		return errors.Wrap(err, "failed to open voice UDP connection")
	}

	// Get the session description from the voice gateway.
	d, err := s.gateway.SessionDescriptionCtx(ctx, voicegateway.SelectProtocol{
		Protocol: "udp",
		Data: voicegateway.SelectProtocolData{
			Address: s.voiceUDP.GatewayIP,
			Port:    s.voiceUDP.GatewayPort,
			Mode:    Protocol,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to select protocol")
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
			return errors.Wrapf(err, "failed to send frame %d", i)
		}
	}
	return nil
}

// Write writes into the UDP voice connection WITHOUT a timeout.
func (s *Session) Write(b []byte) (int, error) {
	return s.WriteCtx(context.Background(), b)
}

// WriteCtx writes into the UDP voice connection with a context for timeout.
func (s *Session) WriteCtx(ctx context.Context, b []byte) (int, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if s.voiceUDP == nil {
		return 0, ErrCannotSend
	}

	return s.voiceUDP.WriteCtx(ctx, b)
}

func (s *Session) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.DisconnectCtx(ctx)
}

func (s *Session) DisconnectCtx(ctx context.Context) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// If we're already closed.
	if s.gateway == nil && s.voiceUDP == nil {
		return nil
	}

	// Notify Discord that we're leaving. This will send a
	// VoiceStateUpdateEvent, in which our handler will promptly remove the
	// session from the map.

	err := s.session.Gateway.UpdateVoiceStateCtx(ctx, gateway.UpdateVoiceStateData{
		GuildID:   s.state.GuildID,
		ChannelID: discord.ChannelID(discord.NullSnowflake),
		SelfMute:  true,
		SelfDeaf:  true,
	})

	s.ensureClosed()
	// wrap returns nil if err is nil
	return errors.Wrap(err, "failed to update voice state")
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
