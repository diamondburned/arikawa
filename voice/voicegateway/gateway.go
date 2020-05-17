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

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/moreatomic"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/pkg/errors"
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
	GuildID   discord.Snowflake
	ChannelID discord.Snowflake
	UserID    discord.Snowflake

	SessionID string
	Token     string
	Endpoint  string
}

// Gateway represents a Discord Gateway Gateway connection.
type Gateway struct {
	state State // constant

	mutex sync.RWMutex
	ready ReadyEvent

	ws *wsutil.Websocket

	Timeout   time.Duration
	reconnect moreatomic.Bool

	EventLoop *wsutil.PacemakerLoop

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
	return &Gateway{
		state:      state,
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

// Open shouldn't be used, but JoinServer instead.
func (c *Gateway) Open() error {
	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	var endpoint = "wss://" + strings.TrimSuffix(c.state.Endpoint, ":80") + "/?v=" + Version

	wsutil.WSDebug("Connecting to voice endpoint (endpoint=" + endpoint + ")")
	c.ws = wsutil.New(endpoint)

	// Create a new context with a timeout for the connection.
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// Connect to the Gateway Gateway.
	if err := c.ws.Dial(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to voice gateway")
	}

	wsutil.WSDebug("Trying to start...")

	// Try to start or resume the connection.
	if err := c.start(); err != nil {
		return err
	}

	return nil
}

// Start .
func (c *Gateway) start() error {
	if err := c.__start(); err != nil {
		wsutil.WSDebug("Start failed: ", err)

		// Close can be called with the mutex still acquired here, as the
		// pacemaker hasn't started yet.
		if err := c.Close(); err != nil {
			wsutil.WSDebug("Failed to close after start fail: ", err)
		}
		return err
	}

	return nil
}

// this function blocks until READY.
func (c *Gateway) __start() error {
	// Make a new WaitGroup for use in background loops:
	c.waitGroup = new(sync.WaitGroup)

	ch := c.ws.Listen()

	// Wait for hello.
	wsutil.WSDebug("Waiting for Hello..")

	var hello *HelloEvent
	_, err := wsutil.AssertEvent(<-ch, HelloOP, &hello)
	if err != nil {
		return errors.Wrap(err, "error at Hello")
	}

	wsutil.WSDebug("Received Hello")

	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	// Turns out Hello is sent right away on connection start.
	if !c.reconnect.Get() {
		if err := c.Identify(); err != nil {
			return errors.Wrap(err, "failed to identify")
		}
	} else {
		if err := c.Resume(); err != nil {
			return errors.Wrap(err, "failed to resume")
		}
	}
	// This bool is because we should only try and Resume once.
	c.reconnect.Set(false)

	// Wait for either Ready or Resumed.
	err = wsutil.WaitForEvent(c, ch, func(op *wsutil.OP) bool {
		return op.Code == ReadyOP || op.Code == ResumedOP
	})
	if err != nil {
		return errors.Wrap(err, "failed to wait for Ready or Resumed")
	}

	// Create an event loop executor.
	c.EventLoop = wsutil.NewLoop(hello.HeartbeatInterval.Duration(), ch, c)

	// Start the event handler, which also handles the pacemaker death signal.
	c.waitGroup.Add(1)

	c.EventLoop.RunAsync(func(err error) {
		c.waitGroup.Done() // mark so Close() can exit.
		wsutil.WSDebug("Event loop stopped.")

		if err != nil {
			c.ErrorLog(err)
			c.Reconnect()
			// Reconnect should spawn another eventLoop in its Start function.
		}
	})

	wsutil.WSDebug("Started successfully.")

	return nil
}

// Close .
func (c *Gateway) Close() error {
	// Check if the WS is already closed:
	if c.waitGroup == nil && c.EventLoop.Stopped() {
		wsutil.WSDebug("Gateway is already closed.")

		c.AfterClose(nil)
		return nil
	}

	// If the pacemaker is running:
	if !c.EventLoop.Stopped() {
		wsutil.WSDebug("Stopping pacemaker...")

		// Stop the pacemaker and the event handler
		c.EventLoop.Stop()

		wsutil.WSDebug("Stopped pacemaker.")
	}

	wsutil.WSDebug("Waiting for WaitGroup to be done.")

	// This should work, since Pacemaker should signal its loop to stop, which
	// would also exit our event loop. Both would be 2.
	c.waitGroup.Wait()

	// Mark g.waitGroup as empty:
	c.waitGroup = nil

	wsutil.WSDebug("WaitGroup is done. Closing the websocket.")

	err := c.ws.Close()
	c.AfterClose(err)
	return err
}

func (c *Gateway) Reconnect() error {
	wsutil.WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	c.Close()

	c.reconnect.Set(true)

	// Condition: err == ErrInvalidSession:
	// If the connection is rate limited (documented behavior):
	// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

	if err := c.Open(); err != nil {
		return errors.Wrap(err, "failed to reopen gateway")
	}

	wsutil.WSDebug("Reconnected successfully.")

	return nil
}

func (c *Gateway) SessionDescription(sp SelectProtocol) (*SessionDescriptionEvent, error) {
	// Add the handler first.
	ch, cancel := c.EventLoop.Extras.Add(func(op *wsutil.OP) bool {
		return op.Code == SessionDescriptionOP
	})
	defer cancel()

	if err := c.SelectProtocol(sp); err != nil {
		return nil, err
	}

	var sesdesc *SessionDescriptionEvent

	// Wait for SessionDescriptionOP packet.
	if err := (<-ch).UnmarshalData(&sesdesc); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal session description")
	}

	return sesdesc, nil
}

// Send .
func (c *Gateway) Send(code OPCode, v interface{}) error {
	return c.send(code, v)
}

// send .
func (c *Gateway) send(code OPCode, v interface{}) error {
	if c.ws == nil {
		return errors.New("tried to send data to a connection without a Websocket")
	}

	if c.ws.Conn == nil {
		return errors.New("tried to send data to a connection with a closed Websocket")
	}

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
	return c.ws.Send(b)
}
