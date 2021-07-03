package voice

import (
	"context"
	"log"
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

// gatewaySession is an interface that both State and Session somewhat
// implements.
type gatewaySession interface {
	Channel(discord.ChannelID) (*discord.Channel, error)
}

// Session is a single voice session that wraps around the voice gateway and UDP
// connection.
type Session struct {
	*handler.Handler
	ErrorLog func(err error)

	upperSession gatewaySession
	upperGateway *gateway.Gateway

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
	voiceUDP *udp.Manager
}

// NewSession creates a new voice session for the current user.
func NewSession(state *state.State) (*Session, error) {
	u, err := state.Me()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get me")
	}

	s := NewSessionCustom(state.Session, u.ID)
	s.upperSession = state

	return s, nil
}

// NewSessionCustom creates a new voice session from the given session and user
// ID.
func NewSessionCustom(ses *session.Session, userID discord.UserID) *Session {
	handler := handler.New()
	hlooper := handleloop.NewLoop(handler)
	session := &Session{
		ErrorLog: func(err error) {},
		Handler:  handler,
		looper:   hlooper,

		upperSession: ses,
		upperGateway: ses.Gateway,

		state: voicegateway.State{
			UserID: userID,
		},
		incoming: make(chan struct{}, 2),
		voiceUDP: udp.NewManager(),
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
	log.Printf("received %#v", ev)

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

// JoinChannel joins a voice channel with a default timeout.
func (s *Session) JoinChannel(chID discord.ChannelID, mute, deaf bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	return s.JoinChannelCtx(ctx, chID, mute, deaf)
}

// JoinChannelCtx joins a voice channel using the given context.
func (s *Session) JoinChannelCtx(
	ctx context.Context, chID discord.ChannelID, mute, deaf bool) error {

	if s.joining.Get() {
		return ErrAlreadyConnecting
	}

	guildID := discord.NullGuildID

	if chID.IsValid() {
		// Validate the channel ID but don't actually check the channel type,
		// since Discord might add more types of channels.
		c, err := s.upperSession.Channel(chID)
		if err != nil {
			return err
		}
		guildID = c.GuildID
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
	s.state.ChannelID = chID
	s.state.GuildID = guildID

	// Ensure that if `cID` is zero that it passes null to the update event.
	if !chID.IsValid() {
		chID = discord.NullChannelID
	}

	// https://discord.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	err := s.upperGateway.UpdateVoiceStateCtx(ctx, gateway.UpdateVoiceStateData{
		GuildID:   guildID,
		ChannelID: chID,
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

	wsutil.WSDebug("Ensure the voice gateway is already gone.")
	if s.gateway != nil {
		s.gateway.Close()
	}

	if s.state.Endpoint == "" {
		// Discord is trying to hand us an endpoint. Whatever, bail.
		wsutil.WSDebug("Empty endpoint received.")
		s.gateway = nil
		// Leave the UDP connection paused.
		return
	}

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
	udpConn, err := s.voiceUDP.PauseAndDial(ctx, voiceReady.Addr(), voiceReady.SSRC)
	if err != nil {
		return errors.Wrap(err, "failed to open voice UDP connection")
	}

	// Resume the UDP connection only once we're done.
	defer s.voiceUDP.Continue()

	// Get the session description from the voice gateway.
	d, err := s.gateway.SessionDescriptionCtx(ctx, voicegateway.SelectProtocol{
		Protocol: "udp",
		Data: voicegateway.SelectProtocolData{
			Address: udpConn.GatewayIP,
			Port:    udpConn.GatewayPort,
			Mode:    Protocol,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to select protocol")
	}

	log.Println("prep to use secret key")
	time.Sleep(2 * time.Second)
	udpConn.UseSecret(d.SecretKey)

	log.Println("secret key used")

	s.voiceUDP.Continue()
	log.Println("UDP resumed")

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

// VoiceUDPManager gets the internal voice UDP connection manager. The caller
// could use this method to change the settings, though this should be done
// preferably before any channels are joined.
func (s *Session) VoiceUDPManager() *udp.Manager {
	return s.voiceUDP
}

// Write writes into the UDP voice connection WITHOUT a timeout. Refer to
// WriteCtx for more information.
func (s *Session) Write(b []byte) (int, error) {
	return s.voiceUDP.Write(b)
}

// LeaveOnCtx is a helper function that leaves the session once the context
// expires.
func (s *Session) LeaveOnCtx(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.Leave()
	}()
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

	// Ensure that we always clean up the resources even when things fail.
	defer s.ensureClosed()

	if s.gateway != nil {
		// Stop the gateway event loop first.
		s.looper.Stop()

		// Notify Discord that we're leaving. This will send a
		// VoiceStateUpdateEvent, in which our handler will promptly remove the
		// session from the map.
		err := s.upperGateway.UpdateVoiceStateCtx(ctx, gateway.UpdateVoiceStateData{
			GuildID:   s.state.GuildID,
			ChannelID: discord.ChannelID(discord.NullSnowflake),
			SelfMute:  true,
			SelfDeaf:  true,
		})

		// wrap returns nil if err is nil
		return errors.Wrap(err, "failed to update voice state")
	}

	return nil
}

// close ensures everything is closed. It does not acquire the mutex.
func (s *Session) ensureClosed() {
	s.looper.Stop()

	// Disconnect the UDP connection.
	s.voiceUDP.Close()

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
	return s.voiceUDP.ReadPacket()
}
