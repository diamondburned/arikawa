package voice

import (
	"context"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/handler"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/handleloop"
	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/diamondburned/arikawa/v3/voice/udp"
	"github.com/diamondburned/arikawa/v3/voice/voicegateway"
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
	looper  *handleloop.Loop
	detach  func()

	mut   sync.RWMutex
	state voicegateway.State // guarded except UserID
	// TODO: expose getters mutex-guarded.
	gateway  *voicegateway.Gateway
	voiceUDP *udp.Connection
	// end of mutex

	WSTimeout      time.Duration // global WSTimeout
	WSMaxRetry     int           // 2
	WSRetryDelay   time.Duration // 2s
	WSWaitDuration time.Duration // 5s

	// joining determines the behavior of incoming event callbacks (Update).
	// If this is true, incoming events will just send into Updated channels. If
	// false, events will trigger a reconnection.
	joining   moreatomic.Bool
	connected bool
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
		ErrorLog:       func(err error) {},
		WSTimeout:      WSTimeout,
		WSMaxRetry:     2,
		WSRetryDelay:   2 * time.Second,
		WSWaitDuration: 5 * time.Second,
	}

	return session
}

// updateServer is specifically used to monitor for reconnects.
func (s *Session) updateServer(ev *gateway.VoiceServerUpdateEvent) {
	if s.joining.Get() {
		return
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	// Ignore if we haven't connected yet or we're still joining.
	if !s.connected || s.state.GuildID != ev.GuildID {
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

// JoinChannel joins a voice channel with the default WS timeout. See
// JoinChannelCtx for more information.
func (s *Session) JoinChannel(gID discord.GuildID, cID discord.ChannelID, mute, deaf bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.JoinChannelCtx(ctx, gID, cID, mute, deaf)
}

type waitEventChs struct {
	serverUpdate chan *gateway.VoiceServerUpdateEvent
	stateUpdate  chan *gateway.VoiceStateUpdateEvent
}

// JoinChannelCtx joins a voice channel. Callers shouldn't use this method
// directly, but rather Voice's. This method shouldn't ever be called
// concurrently.
func (s *Session) JoinChannelCtx(
	ctx context.Context, gID discord.GuildID, cID discord.ChannelID, mute, deaf bool) error {

	s.mut.Lock()
	defer s.mut.Unlock()

	// Error out if we're already joining. JoinChannel shouldn't be called
	// concurrently.
	if s.joining.Get() {
		return errors.New("JoinChannel working elsewhere")
	}

	s.joining.Set(true)
	defer s.joining.Set(false)

	// Set the state.
	s.state.ChannelID = cID
	s.state.GuildID = gID
	s.detach = s.session.AddHandler(s.updateServer)

	// Ensure that if `cID` is zero that it passes null to the update event.
	channelID := discord.NullChannelID
	if cID.IsValid() {
		channelID = cID
	}

	chs := waitEventChs{
		serverUpdate: make(chan *gateway.VoiceServerUpdateEvent),
		stateUpdate:  make(chan *gateway.VoiceStateUpdateEvent),
	}

	// Bind the handlers.
	cancels := []func(){
		s.session.AddHandler(chs.serverUpdate),
		s.session.AddHandler(chs.stateUpdate),
	}
	// Disconnects the handlers once the function exits.
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	// Ensure gateway and voiceUDP are already closed.
	s.ensureClosed()

	data := gateway.UpdateVoiceStateData{
		GuildID:   gID,
		ChannelID: channelID,
		SelfMute:  mute,
		SelfDeaf:  deaf,
	}

	var err error

	var timer *time.Timer

	// Retry 3 times maximum.
	for i := 0; i < s.WSMaxRetry; i++ {
		if err = s.askDiscord(ctx, data, chs); err == nil {
			break
		}

		// If this is the first attempt and the context timed out, it's
		// probably the context that's waiting for gateway events. Retry the
		// loop.
		if i == 0 && errors.Is(err, ctx.Err()) {
			continue
		}

		if timer == nil {
			// Set up a timer.
			timer = time.NewTimer(s.WSRetryDelay)
			defer timer.Stop()
		} else {
			timer.Reset(s.WSRetryDelay)
		}

		select {
		case <-timer.C:
			continue
		case <-ctx.Done():
			return err
		}
	}

	// These 2 methods should've updated s.state before sending into these
	// channels. Since s.state is already filled, we can go ahead and connect.

	// Mark the session as connected and move on. This allows one of the
	// connected handlers to reconnect on its own.
	s.connected = true

	return s.reconnectCtx(ctx)
}

func (s *Session) askDiscord(
	ctx context.Context, data gateway.UpdateVoiceStateData, chs waitEventChs) error {

	// https://discord.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	if err := s.session.Gateway.UpdateVoiceStateCtx(ctx, data); err != nil {
		return errors.Wrap(err, "failed to send Voice State Update event")
	}

	// Wait for 2 replies. The above command should reply with these 2 events.
	if err := s.waitForIncoming(ctx, chs); err != nil {
		return errors.Wrap(err, "failed to wait for needed gateway events")
	}

	return nil
}

func (s *Session) waitForIncoming(ctx context.Context, chs waitEventChs) error {
	ctx, cancel := context.WithTimeout(ctx, s.WSWaitDuration)
	defer cancel()

	state := false
	// server is true when we already have the token and endpoint, meaning that
	// we don't have to wait for another such event.
	server := s.state.Token != "" && s.state.Endpoint != ""

	// Loop until timeout or until we have all the information that we need.
	for !(server && state) {
		select {
		case ev := <-chs.serverUpdate:
			if s.state.GuildID != ev.GuildID {
				continue
			}
			s.state.Endpoint = ev.Endpoint
			s.state.Token = ev.Token
			server = true

		case ev := <-chs.stateUpdate:
			if s.state.GuildID != ev.GuildID || s.state.UserID != ev.UserID {
				continue
			}
			s.state.SessionID = ev.SessionID
			s.state.ChannelID = ev.ChannelID
			state = true

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

	s.connected = false

	// Unbind the handlers.
	if s.detach != nil {
		s.detach()
		s.detach = nil
	}

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
