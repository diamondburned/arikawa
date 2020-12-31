//
// For the brave souls who get this far: You are the chosen ones,
// the valiant knights of programming who toil away, without rest,
// fixing our most awful code.  To you, true saviors, kings of men,
// I say this: never gonna give you up, never gonna let you down,
// never gonna run around and desert you.  Never gonna make you cry,
// never gonna say goodbye.  Never gonna tell a lie and hurt you.
//

package voicegateway

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/internal/moreatomic"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/wsutil"
)

const (
	// Version represents the current version of the Discord Gateway Gateway this package uses.
	Version = "4"
)

var (
	ErrNoSessionID = errors.New("no sessionID was received")
	ErrNoEndpoint  = errors.New("no endpoint was received")
)

// State contains state information of a voice gateway.
type State struct {
	GuildID   discord.GuildID
	ChannelID discord.ChannelID
	UserID    discord.UserID

	SessionID string
	Token     string
	Endpoint  string
}

// Gateway represents a Discord Gateway Gateway connection.
type Gateway struct {
	state State // constant

	mutex sync.RWMutex
	ready ReadyEvent

	WS *wsutil.Websocket

	Timeout   time.Duration
	reconnect moreatomic.Bool

	EventLoop wsutil.PacemakerLoop

	// ErrorLog will be called when an error occurs (defaults to log.Println)
	ErrorLog func(err error)
	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions. (defaults to noop)
	AfterClose func(err error)

	// Filled by methods, internal use
	waitGroup *sync.WaitGroup
}

func New(state State) *Gateway {
	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	var endpoint = "wss://" + strings.TrimSuffix(state.Endpoint, ":80") + "/?v=" + Version

	return &Gateway{
		state:      state,
		WS:         wsutil.New(endpoint),
		Timeout:    wsutil.WSTimeout,
		ErrorLog:   wsutil.WSError,
		AfterClose: func(error) {},
	}
}

// TODO: get rid of
func (c *Gateway) Ready() ReadyEvent {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ready
}

// OpenCtx shouldn't be used, but JoinServer instead.
func (c *Gateway) OpenCtx(ctx context.Context) error {
	if c.state.Endpoint == "" {
		return errors.New("missing endpoint in state")
	}

	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	var endpoint = "wss://" + strings.TrimSuffix(c.state.Endpoint, ":80") + "/?v=" + Version

	wsutil.WSDebug("VoiceGateway: Connecting to voice endpoint (endpoint=" + endpoint + ")")

	// Create a new context with a timeout for the connection.
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// Connect to the Gateway Gateway.
	if err := c.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to voice gateway")
	}

	wsutil.WSDebug("VoiceGateway: Trying to start...")

	// Try to start or resume the connection.
	if err := c.start(ctx); err != nil {
		return err
	}

	return nil
}

// Start .
func (c *Gateway) start(ctx context.Context) error {
	if err := c.__start(ctx); err != nil {
		wsutil.WSDebug("VoiceGateway: Start failed: ", err)

		// Close can be called with the mutex still acquired here, as the
		// pacemaker hasn't started yet.
		if err := c.Close(); err != nil {
			wsutil.WSDebug("VoiceGateway: Failed to close after start fail: ", err)
		}
		return err
	}

	return nil
}

// this function blocks until READY.
func (c *Gateway) __start(ctx context.Context) error {
	// Make a new WaitGroup for use in background loops:
	c.waitGroup = new(sync.WaitGroup)

	ch := c.WS.Listen()

	// Wait for hello.
	wsutil.WSDebug("VoiceGateway: Waiting for Hello..")

	var hello *HelloEvent
	// Wait for the Hello event; return if it times out.
	select {
	case e, ok := <-ch:
		if !ok {
			return errors.New("unexpected ws close while waiting for Hello")
		}
		if _, err := wsutil.AssertEvent(e, HelloOP, &hello); err != nil {
			return errors.Wrap(err, "error at Hello")
		}
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "failed to wait for Hello event")
	}

	wsutil.WSDebug("VoiceGateway: Received Hello")

	// Start the event handler, which also handles the pacemaker death signal.
	c.waitGroup.Add(1)

	c.EventLoop.StartBeating(hello.HeartbeatInterval.Duration(), c, func(err error) {
		c.waitGroup.Done() // mark so Close() can exit.
		wsutil.WSDebug("VoiceGateway: Event loop stopped.")

		if err != nil {
			c.ErrorLog(err)

			if err := c.Reconnect(); err != nil {
				c.ErrorLog(errors.Wrap(err, "failed to reconnect voice"))
			}

			// Reconnect should spawn another eventLoop in its Start function.
		}
	})

	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	// Turns out Hello is sent right away on connection start.
	if !c.reconnect.Get() {
		if err := c.IdentifyCtx(ctx); err != nil {
			return errors.Wrap(err, "failed to identify")
		}
	} else {
		if err := c.ResumeCtx(ctx); err != nil {
			return errors.Wrap(err, "failed to resume")
		}
	}
	// This bool is because we should only try and Resume once.
	c.reconnect.Set(false)

	// Wait for either Ready or Resumed.
	err := wsutil.WaitForEvent(ctx, c, ch, func(op *wsutil.OP) bool {
		return op.Code == ReadyOP || op.Code == ResumedOP
	})
	if err != nil {
		return errors.Wrap(err, "failed to wait for Ready or Resumed")
	}

	// Bind the event channel away.
	c.EventLoop.SetEventChannel(ch)

	wsutil.WSDebug("VoiceGateway: Started successfully.")

	return nil
}

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() error {
	wsutil.WSDebug("VoiceGateway: Trying to close. Pacemaker check skipped.")

	wsutil.WSDebug("VoiceGateway: Closing the Websocket...")
	err := g.WS.Close()

	if errors.Is(err, wsutil.ErrWebsocketClosed) {
		wsutil.WSDebug("VoiceGateway: Websocket already closed.")
		return nil
	}

	wsutil.WSDebug("VoiceGateway: Websocket closed; error:", err)

	wsutil.WSDebug("VoiceGateway: Waiting for the Pacemaker loop to exit.")
	g.waitGroup.Wait()
	wsutil.WSDebug("VoiceGateway: Pacemaker loop exited.")

	g.AfterClose(err)
	wsutil.WSDebug("VoiceGateway: AfterClose callback finished.")

	return err
}

func (c *Gateway) Reconnect() error {
	return c.ReconnectCtx(context.Background())
}

func (c *Gateway) ReconnectCtx(ctx context.Context) error {
	wsutil.WSDebug("VoiceGateway: Reconnecting...")

	// TODO: implement a reconnect loop

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	c.Close()

	c.reconnect.Set(true)

	// Condition: err == ErrInvalidSession:
	// If the connection is rate limited (documented behavior):
	// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

	if err := c.OpenCtx(ctx); err != nil {
		return errors.Wrap(err, "failed to reopen gateway")
	}

	wsutil.WSDebug("VoiceGateway: Reconnected successfully.")

	return nil
}

func (c *Gateway) SessionDescriptionCtx(
	ctx context.Context, sp SelectProtocol) (*SessionDescriptionEvent, error) {

	// Add the handler first.
	ch, cancel := c.EventLoop.Extras.Add(func(op *wsutil.OP) bool {
		return op.Code == SessionDescriptionOP
	})
	defer cancel()

	if err := c.SelectProtocolCtx(ctx, sp); err != nil {
		return nil, err
	}

	var sesdesc *SessionDescriptionEvent

	// Wait for SessionDescriptionOP packet.
	select {
	case e, ok := <-ch:
		if !ok {
			return nil, errors.New("unexpected close waiting for session description")
		}
		if err := e.UnmarshalData(&sesdesc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal session description")
		}
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "failed to wait for session description")
	}

	return sesdesc, nil
}

// Send sends a payload to the Gateway with the default timeout.
func (c *Gateway) Send(code OPCode, v interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	return c.SendCtx(ctx, code, v)
}

func (c *Gateway) SendCtx(ctx context.Context, code OPCode, v interface{}) error {
	var op = wsutil.OP{
		Code: code,
	}

	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "failed to encode v")
		}

		op.Data = b
	}

	b, err := json.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "failed to encode payload")
	}

	// WS should already be thread-safe.
	return c.WS.SendCtx(ctx, b)
}
