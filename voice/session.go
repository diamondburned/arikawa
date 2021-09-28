package voice

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/ws/ophandler"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/lazytime"
	"github.com/diamondburned/arikawa/v3/internal/moreatomic"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/ws"
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
const WSTimeout = 25 * time.Second

// ReconnectError is emitted into Session.Handler everytime the voice gateway
// fails to be reconnected. It implements the error interface.
type ReconnectError struct {
	Err error
}

// Error implements error.
func (e ReconnectError) Error() string {
	return "voice reconnect error: " + e.Err.Error()
}

// Unwrap returns e.Err.
func (e ReconnectError) Unwrap() error { return e.Err }

// MainSession abstracts both session.Session and state.State.
type MainSession interface {
	// AddHandler describes the method in handler.Handler.
	AddHandler(handler interface{}) (rm func())
	// Gateway returns the session's main Discord gateway.
	Gateway() *gateway.Gateway
	// Me returns the current user.
	Me() (*discord.User, error)
	// Channel queries for the channel with the given ID.
	Channel(discord.ChannelID) (*discord.Channel, error)
}

var (
	_ MainSession = (*session.Session)(nil)
	_ MainSession = (*state.State)(nil)
)

// UDPDialer is the UDP dialer function type. It's the function signature for
// udp.DialConnection.
type UDPDialer = func(ctx context.Context, addr string, ssrc uint32) (*udp.Connection, error)

// Session is a single voice session that wraps around the voice gateway and UDP
// connection.
type Session struct {
	*handler.Handler
	session MainSession

	mut sync.RWMutex
	// connected is a non-nil blocking channel after Join is called and is
	// closed once Leave is called.
	disconnected chan struct{}

	state voicegateway.State // guarded except UserID

	detachReconnect []func()

	voiceUDP *udp.Connection
	gateway  *voicegateway.Gateway
	gwCancel context.CancelFunc
	gwDone   <-chan struct{}

	// DialUDP is the custom function for dialing up a UDP connection.
	DialUDP UDPDialer

	WSTimeout      time.Duration // global WSTimeout
	WSMaxRetry     int           // 2
	WSRetryDelay   time.Duration // 2s
	WSWaitDuration time.Duration // 5s

	// joining determines the behavior of incoming event callbacks (Update).
	// If this is true, incoming events will just send into Updated channels. If
	// false, events will trigger a reconnection.
	joining moreatomic.Bool
	// disconnectClosed is true if connected is already closed. It is only used
	// to keep track of closing connected.
	disconnectClosed bool
}

// NewSession creates a new voice session for the current user.
func NewSession(state MainSession) (*Session, error) {
	u, err := state.Me()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get me")
	}

	return NewSessionCustom(state, u.ID), nil
}

// NewSessionCustom creates a new voice session from the given session and user
// ID.
func NewSessionCustom(ses MainSession, userID discord.UserID) *Session {
	closed := make(chan struct{})
	close(closed)

	session := &Session{
		Handler: handler.New(),
		session: ses,
		state: voicegateway.State{
			UserID: userID,
		},
		DialUDP:        udp.DialConnection,
		WSTimeout:      WSTimeout,
		WSMaxRetry:     2,
		WSRetryDelay:   2 * time.Second,
		WSWaitDuration: 5 * time.Second,

		// Set this pair of value in so we never have to nil-check the channel.
		// We can just assume that it's either closed or connected.
		disconnected:     closed,
		disconnectClosed: true,
	}

	return session
}

func (s *Session) acquireUpdate(f func()) bool {
	if s.joining.Get() {
		return false
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	// Ignore if we haven't connected yet or we're still joining.
	select {
	case <-s.disconnected:
		return false
	default:
		// ok
	}

	f()
	return true
}

// updateServer is specifically used to monitor for reconnects.
func (s *Session) updateServer(ev *gateway.VoiceServerUpdateEvent) {
	s.acquireUpdate(func() {
		if s.state.GuildID != ev.GuildID {
			return
		}

		s.state.Endpoint = ev.Endpoint
		s.state.Token = ev.Token

		ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
		defer cancel()

		s.reconnectCtx(ctx)
	})
}

// updateState is specifically used after connecting to monitor when the bot is
// forced across channels.
func (s *Session) updateState(ev *gateway.VoiceStateUpdateEvent) {
	s.acquireUpdate(func() {
		if s.state.GuildID != ev.GuildID || s.state.UserID != ev.UserID {
			return
		}

		s.state.ChannelID = ev.ChannelID
		s.state.SessionID = ev.SessionID

		ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
		defer cancel()

		s.reconnectCtx(ctx)
	})
}

type waitEventChs struct {
	serverUpdate chan *gateway.VoiceServerUpdateEvent
	stateUpdate  chan *gateway.VoiceStateUpdateEvent
}

// JoinChannel joins the given voice channel with the default timeout.
func (s *Session) JoinChannel(ctx context.Context, chID discord.ChannelID, mute, deaf bool) error {
	var ch *discord.Channel

	if chID.IsValid() {
		var err error
		ch, err = s.session.Channel(chID)
		if err != nil {
			return errors.Wrap(err, "invalid channel ID")
		}
	}

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
	if ch != nil {
		s.state.ChannelID = ch.ID
		s.state.GuildID = ch.GuildID
	} else {
		s.state.GuildID = 0
		// Ensure that if `cID` is zero that it passes null to the update event.
		s.state.ChannelID = discord.NullChannelID
	}

	if s.detachReconnect == nil {
		s.detachReconnect = []func(){
			s.session.AddHandler(s.updateServer),
			s.session.AddHandler(s.updateState),
		}
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

	// https://discord.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	data := &gateway.UpdateVoiceStateCommand{
		GuildID:   s.state.GuildID,
		ChannelID: s.state.ChannelID,
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
	s.disconnected = make(chan struct{})

	return s.reconnectCtx(ctx)
}

func (s *Session) askDiscord(
	ctx context.Context, data *gateway.UpdateVoiceStateCommand, chs waitEventChs) error {

	// https://discord.com/developers/docs/topics/voice-connections#retrieving-voice-server-information
	// Send a Voice State Update event to the gateway.
	if err := s.session.Gateway().Send(ctx, data); err != nil {
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
func (s *Session) reconnectCtx(ctx context.Context) error {
	ws.WSDebug("Sending stop handle.")

	s.ensureClosed()

	ws.WSDebug("Start gateway.")
	s.gateway = voicegateway.New(s.state)

	// Open the voice gateway. The function will block until Ready is received.
	gwctx, gwcancel := context.WithCancel(context.Background())
	s.gwCancel = gwcancel

	gwch := s.gateway.Connect(gwctx)

	if err := s.spinGateway(ctx, gwch); err != nil {
		// Early cancel the gateway.
		gwcancel()
		// Nil this so future reconnects don't use the invalid gwDone.
		s.gwCancel = nil
		// Emit the error. It's fine to do this here since this is the only
		// place that can error out.
		s.Handler.Call(&ReconnectError{err})
		return err
	}

	// Start dispatching.
	s.gwDone = ophandler.Loop(gwch, s.Handler)

	return nil
}

func (s *Session) spinGateway(ctx context.Context, gwch <-chan ws.Op) error {
	var err error

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-gwch:
			if !ok {
				return s.gateway.LastError()
			}

			switch data := ev.Data.(type) {
			case *ws.CloseEvent:
				return errors.Wrap(err, "voice gateway error")

			case *voicegateway.ReadyEvent:
				// Prepare the UDP voice connection.
				s.voiceUDP, err = s.DialUDP(ctx, data.Addr(), data.SSRC)
				if err != nil {
					return errors.Wrap(err, "failed to open voice UDP connection")
				}

				if err := s.gateway.Send(ctx, &voicegateway.SelectProtocolCommand{
					Protocol: "udp",
					Data: voicegateway.SelectProtocolData{
						Address: s.voiceUDP.GatewayIP,
						Port:    s.voiceUDP.GatewayPort,
						Mode:    Protocol,
					},
				}); err != nil {
					return errors.Wrap(err, "failed to send SelectProtocolCommand")
				}

			case *voicegateway.SessionDescriptionEvent:
				// We're done.
				s.voiceUDP.UseSecret(data.SecretKey)
				return nil
			}

			// Dispatch this event to the handler.
			s.Handler.Call(ev.Data)
		}
	}
}

// Speaking tells Discord we're speaking. This method should not be called
// concurrently.
func (s *Session) Speaking(ctx context.Context, flag voicegateway.SpeakingFlag) error {
	s.mut.Lock()
	gateway := s.gateway
	s.mut.Unlock()

	return gateway.Speaking(ctx, flag)
}

func (s *Session) useUDP(f func(c *udp.Connection) error) (err error) {
	const maxAttempts = 5
	const retryDelay = 250 * time.Millisecond // adds up to about 1.25s

	var lazyWait lazytime.Timer

	// Hack: loop until we no longer get an error closed or until the connection
	// is dead. This is a workaround for when the session is trying to reconnect
	// itself in the background, which would drop the UDP connection.
	for i := 0; i < maxAttempts; i++ {
		s.mut.RLock()
		voiceUDP := s.voiceUDP
		disconnected := s.disconnected
		s.mut.RUnlock()

		select {
		case <-disconnected:
			return net.ErrClosed
		default:
			if voiceUDP == nil {
				// Session is still connected, but our voice UDP connection is
				// nil, so we're probably in the process of reconnecting
				// already.
				goto retry
			}
		}

		if err = f(voiceUDP); err != nil && errors.Is(err, net.ErrClosed) {
			// Session is still connected, but our UDP connection is somehow
			// closed, so we're probably waiting for the server to ask us to
			// reconnect with a new session.
			goto retry
		}

		// Unknown error or none at all; exit.
		return err

	retry:
		// Wait a slight bit. We can probably make the caller wait a couple
		// milliseconds without a wait.
		lazyWait.Reset(retryDelay)
		select {
		case <-lazyWait.C:
			continue
		case <-disconnected:
			return net.ErrClosed
		}
	}

	return
}

// Write writes into the UDP voice connection. This method is thread safe as far
// as calling other methods of Session goes; HOWEVER it is not thread safe to
// call Write itself concurrently.
func (s *Session) Write(b []byte) (int, error) {
	var n int
	err := s.useUDP(func(c *udp.Connection) (err error) {
		n, err = c.Write(b)
		return
	})
	return n, err
}

// ReadPacket reads a single packet from the UDP connection. This is NOT at all
// thread safe, and must be used very carefully. The backing buffer is always
// reused.
func (s *Session) ReadPacket() (*udp.Packet, error) {
	var p *udp.Packet
	err := s.useUDP(func(c *udp.Connection) (err error) {
		p, err = c.ReadPacket()
		return
	})
	return p, err
}

// Leave disconnects the current voice session from the currently connected
// channel.
func (s *Session) Leave(ctx context.Context) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.ensureClosed()

	// Unbind the handlers.
	if s.detachReconnect != nil {
		for _, detach := range s.detachReconnect {
			detach()
		}
		s.detachReconnect = nil
	}

	// If we're already closed.
	if s.gateway == nil && s.voiceUDP == nil {
		return nil
	}

	// Notify Discord that we're leaving.
	err := s.session.Gateway().Send(ctx, &gateway.UpdateVoiceStateCommand{
		GuildID:   s.state.GuildID,
		ChannelID: discord.ChannelID(discord.NullSnowflake),
		SelfMute:  true,
		SelfDeaf:  true,
	})

	// Wait for the gateway to exit first before we tell the user of the gateway
	// send error.
	if err := s.cancelGateway(ctx); err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "failed to update voice state")
	}

	return nil
}

func (s *Session) cancelGateway(ctx context.Context) error {
	if s.gwCancel != nil {
		s.gwCancel()
		s.gwCancel = nil

		// Wait for the previous gateway to finish closing up, but make sure to
		// bail if the context expires.
		if err := ophandler.WaitForDone(ctx, s.gwDone); err != nil {
			return errors.Wrap(err, "cannot wait for gateway to close")
		}
	}

	return nil
}

// close ensures everything is closed. It does not acquire the mutex.
func (s *Session) ensureClosed() {
	// Disconnect the UDP connection.
	if s.voiceUDP != nil {
		s.voiceUDP.Close()
		s.voiceUDP = nil
	}

	if !s.disconnectClosed {
		close(s.disconnected)
		s.disconnectClosed = true
	}

	if s.gwCancel != nil {
		s.gwCancel()
		// Don't actually clear this field, because we still want the caller to
		// be able to wait for the gateway to completely exit using
		// cancelGateway.
	}
}
