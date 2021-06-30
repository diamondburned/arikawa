package voice

import (
	"context"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v2/state"
	"github.com/diamondburned/arikawa/v2/utils/handler"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/internal/handleloop"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/diamondburned/arikawa/v2/voice/udp"
	"github.com/diamondburned/arikawa/v2/voice/voicegateway"
)

// Protocol is the encryption protocol that this library uses.
const Protocol = "xsalsa20_poly1305"

// ErrAlreadyConnecting is returned when the session is already connecting.
var ErrAlreadyConnecting = errors.New("already connecting")

// ErrCannotSend is an error when audio is sent to a closed channel.
var ErrCannotSend = errors.New("cannot send audio to closed channel")

// WSTimeout is the duration to wait for a gateway operation including Session
// to complete before erroring out. This only applies to functions that don't
// take in a context already.
var WSTimeout = 10 * time.Second

// Session is a single voice session that wraps around the voice gateway and UDP
// connection.
type Session struct {
	*handler.Handler
	ErrorLog func(err error)

	session *session.Session
	cancels []func()
	looper  *handleloop.Loop

	// joining determines the behavior of incoming event callbacks (Update).
	// If this is true, incoming events will just send into Updated channels. If
	// false, events will trigger a reconnection.
	joining  moreatomic.Bool
	incoming chan struct{} // used only when joining == true

	mut sync.RWMutex

	state voicegateway.State // guarded except UserID

	// TODO: expose getters mutex-guarded.
	gateway  *voicegateway.Gateway
	voiceUDP *udp.Connection
}

// NewSession creates a new voice session for the current user.
func NewSession(state *state.State) (*Session, error) {
	u, err := state.Me()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get me")
	}

	return NewSessionCustom(state.Session, u.ID), nil
}

// NewSessionCustom creates a new voice session from the given session and user
// ID.
func NewSessionCustom(ses *session.Session, userID discord.UserID) *Session {
	handler := handler.New()
	hlooper := handleloop.NewLoop(handler)
	session := &Session{
		Handler: handler,
		looper:  hlooper,
		session: ses,
		state: voicegateway.State{
			UserID: userID,
		},
		ErrorLog: func(err error) {},
		incoming: make(chan struct{}, 2),
	}
	session.cancels = []func(){
		ses.AddHandler(session.updateServer),
		ses.AddHandler(session.updateState),
	}

	return session
}

func (s *Session) updateServer(ev *gateway.VoiceServerUpdateEvent) {
	// If this is true, then mutex is acquired already.
	if s.joining.Get() {
		if s.state.GuildID != ev.GuildID {
			return
		}

		s.state.Endpoint = ev.Endpoint
		s.state.Token = ev.Token

		s.incoming <- struct{}{}
		return
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	if s.state.GuildID != ev.GuildID {
		return
	}

	// Reconnect.

	s.state.Endpoint = ev.Endpoint
	s.state.Token = ev.Token

	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	if err := s.reconnectCtx(ctx); err != nil {
		s.ErrorLog(errors.Wrap(err, "failed to reconnect after voice server update"))
	}
}

func (s *Session) updateState(ev *gateway.VoiceStateUpdateEvent) {
	if s.state.UserID != ev.UserID { // constant so no mutex
		// Not our state.
		return
	}

	// If this is true, then mutex is acquired already.
	if s.joining.Get() {
		if s.state.GuildID != ev.GuildID {
			return
		}

		s.state.SessionID = ev.SessionID
		s.state.ChannelID = ev.ChannelID

		s.incoming <- struct{}{}
		return
	}
}

func (s *Session) JoinChannel(gID discord.GuildID, cID discord.ChannelID, mute, deaf bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.JoinChannelCtx(ctx, gID, cID, mute, deaf)
}

// JoinChannelCtx joins a voice channel. Callers shouldn't use this method
// directly, but rather Voice's. This method shouldn't ever be called
// concurrently.
func (s *Session) JoinChannelCtx(
	ctx context.Context, gID discord.GuildID, cID discord.ChannelID, mute, deaf bool) error {

	if s.joining.Get() {
		return ErrAlreadyConnecting
	}

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

	// Ensure that if `cID` is zero that it passes null to the update event.
	channelID := discord.NullChannelID
	if cID.IsValid() {
		channelID = cID
	}

	// https://discord.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	err := s.session.Gateway.UpdateVoiceStateCtx(ctx, gateway.UpdateVoiceStateData{
		GuildID:   gID,
		ChannelID: channelID,
		SelfMute:  mute,
		SelfDeaf:  deaf,
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
	wsutil.WSDebug("Sending stop handle.")
	s.looper.Stop()

	wsutil.WSDebug("Start gateway.")
	s.gateway = voicegateway.New(s.state)

	// Open the voice gateway. The function will block until Ready is received.
	if err := s.gateway.OpenCtx(ctx); err != nil {
		return errors.Wrap(err, "failed to open voice gateway")
	}

	// Start the handler dispatching
	s.looper.Start(s.gateway.Events)

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

	s.voiceUDP.UseSecret(d.SecretKey)

	return nil
}

// Speaking tells Discord we're speaking. This method should not be called
// concurrently.
func (s *Session) Speaking(flag voicegateway.SpeakingFlag) error {
	s.mut.RLock()
	gateway := s.gateway
	s.mut.RUnlock()

	return gateway.Speaking(flag)
}

// UseContext tells the UDP voice connection to write with the given context.
func (s *Session) UseContext(ctx context.Context) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if s.voiceUDP == nil {
		return ErrCannotSend
	}

	return s.voiceUDP.UseContext(ctx)
}

// VoiceUDPConn gets a voice UDP connection. The caller could use this method to
// circumvent the rapid mutex-read-lock acquire inside Write.
func (s *Session) VoiceUDPConn() *udp.Connection {
	s.mut.RLock()
	defer s.mut.RUnlock()

	return s.voiceUDP
}

// Write writes into the UDP voice connection WITHOUT a timeout. Refer to
// WriteCtx for more information.
func (s *Session) Write(b []byte) (int, error) {
	return s.WriteCtx(context.Background(), b)
}

// WriteCtx writes into the UDP voice connection with a context for timeout.
// This method is thread safe as far as calling other methods of Session goes;
// HOWEVER it is not thread safe to call Write itself concurrently.
func (s *Session) WriteCtx(ctx context.Context, b []byte) (int, error) {
	voiceUDP := s.VoiceUDPConn()

	if voiceUDP == nil {
		return 0, ErrCannotSend
	}

	return voiceUDP.WriteCtx(ctx, b)
}

// Leave disconnects the current voice session from the currently connected
// channel.
func (s *Session) Leave() error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.LeaveCtx(ctx)
}

// LeaveCtx disconencts with a context. Refer to Leave for more information.
func (s *Session) LeaveCtx(ctx context.Context) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	// If we're already closed.
	if s.gateway == nil && s.voiceUDP == nil {
		return nil
	}

	s.looper.Stop()

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
	s.looper.Stop()

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

// ReadPacket reads a single packet from the UDP connection. This is NOT at all
// thread safe, and must be used very carefully. The backing buffer is always
// reused.
func (s *Session) ReadPacket() (*udp.Packet, error) {
	return s.VoiceUDPConn().ReadPacket()
}
