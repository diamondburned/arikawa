//
// For the brave souls who get this far: You are the chosen ones,
// the valiant knights of programming who toil away, without rest,
// fixing our most awful code.  To you, true saviors, kings of men,
// I say this: never gonna give you up, never gonna let you down,
// never gonna run around and desert you.  Never gonna make you cry,
// never gonna say goodbye.  Never gonna tell a lie and hurt you.
//

package voice

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/pkg/errors"
)

var (
	ErrNoSessionID = errors.New("no SessionID was received after 1 second")
)

// Connection represents a Discord Voice Gateway connection.
type Connection struct {
	json.Driver
	mut sync.RWMutex

	SessionID    string
	Token        string
	Endpoint     string
	reconnection bool

	UserID    discord.Snowflake
	GuildID   discord.Snowflake
	ChannelID discord.Snowflake

	muted    bool
	deafened bool
	speaking bool

	WS        *wsutil.Websocket
	WSTimeout time.Duration

	Pacemaker *gateway.Pacemaker

	udpConn  *net.UDPConn
	OpusSend chan []byte
	close    chan struct{}

	// ErrorLog will be called when an error occurs (defaults to log.Println)
	ErrorLog func(err error)
	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions. (defaults to noop)
	AfterClose func(err error)

	// Stored Operations
	hello              HelloEvent
	ready              ReadyEvent
	sessionDescription SessionDescriptionEvent

	// Operation Channels
	helloChan       chan bool
	readyChan       chan bool
	sessionDescChan chan bool

	// Filled by methods, internal use
	paceDeath chan error
	waitGroup *sync.WaitGroup
}

// newConnection .
func newConnection() *Connection {
	return &Connection{
		Driver: json.Default{},

		WSTimeout: WSTimeout,

		close: make(chan struct{}),

		ErrorLog:   defaultErrorHandler,
		AfterClose: func(error) {},

		helloChan:       make(chan bool),
		readyChan:       make(chan bool),
		sessionDescChan: make(chan bool),
	}
}

// Open .
func (c *Connection) Open() error {
	// Having this acquire a lock might cause a problem if the `onVoiceStateUpdate`
	// does not set a session id in time :/
	c.mut.Lock()
	defer c.mut.Unlock()

	// Check if the connection already has a websocket.
	if c.WS != nil {
		WSDebug("Connection already has an active websocket")
		return nil
	}

	// I doubt this would happen from my testing, but you never know.
	if c.SessionID == "" {
		return ErrNoSessionID
	}
	WSDebug("Connection has a session id")

	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	endpoint := "wss://" + strings.TrimSuffix(c.Endpoint, ":80") + "/?v=" + Version

	WSDebug("Connecting to voice endpoint (endpoint=" + endpoint + ")")

	c.WS = wsutil.NewCustom(wsutil.NewConn(c.Driver), endpoint)

	// Create a new context with a timeout for the connection.
	ctx, cancel := context.WithTimeout(context.Background(), c.WSTimeout)
	defer cancel()

	// Connect to the Voice Gateway.
	if err := c.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "Failed to connect to Voice Gateway")
	}

	WSDebug("Trying to start...")

	// Try to resume the connection
	if err := c.Start(); err != nil {
		return err
	}

	return nil
}

// Start .
func (c *Connection) Start() error {
	if err := c.start(); err != nil {
		WSDebug("Start failed: ", err)

		// Close can be called with the mutex still acquired here, as the
		// pacemaker hasn't started yet.
		if err := c.Close(); err != nil {
			WSDebug("Failed to close after start fail: ", err)
		}
		return err
	}

	return nil
}

// start .
func (c *Connection) start() error {
	// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection
	// Apparently we send an Identify or Resume once we are connected unlike the other gateway that
	// waits for a Hello then sends an Identify or Resume.
	if !c.reconnection {
		if err := c.Identify(); err != nil {
			return errors.Wrap(err, "Failed to identify")
		}
	} else {
		if err := c.Resume(); err != nil {
			return errors.Wrap(err, "Failed to resume")
		}
	}
	c.reconnection = false

	// Make a new WaitGroup for use in background loops:
	c.waitGroup = new(sync.WaitGroup)

	// Start the websocket handler.
	go c.handleWS()

	// Wait for hello.
	WSDebug("Waiting for Hello..")
	<-c.helloChan
	WSDebug("Received Hello")

	// Start the pacemaker with the heartrate received from Hello, after
	// initializing everything. This ensures we only heartbeat if the websocket
	// is authenticated.
	c.Pacemaker = &gateway.Pacemaker{
		Heartrate: time.Duration(int(c.hello.HeartbeatInterval)) * time.Millisecond,
		Pace:      c.Heartbeat,
	}
	// Pacemaker dies here, only when it's fatal.
	c.paceDeath = c.Pacemaker.StartAsync(c.waitGroup)

	// Start the event handler, which also handles the pacemaker death signal.
	c.waitGroup.Add(1)

	WSDebug("Started successfully.")

	return nil
}

// Close .
func (c *Connection) Close() error {
	if c.udpConn != nil {
		WSDebug("Closing udp connection.")
		close(c.close)
	}

	// Check if the WS is already closed:
	if c.waitGroup == nil && c.paceDeath == nil {
		WSDebug("Gateway is already closed.")

		c.AfterClose(nil)
		return nil
	}

	// If the pacemaker is running:
	if c.paceDeath != nil {
		WSDebug("Stopping pacemaker...")

		// Stop the pacemaker and the event handler
		c.Pacemaker.Stop()

		WSDebug("Stopped pacemaker.")
	}

	WSDebug("Waiting for WaitGroup to be done.")

	// This should work, since Pacemaker should signal its loop to stop, which
	// would also exit our event loop. Both would be 2.
	c.waitGroup.Wait()

	// Mark g.waitGroup as empty:
	c.waitGroup = nil

	WSDebug("WaitGroup is done. Closing the websocket.")

	err := c.WS.Close()
	c.AfterClose(err)
	return err
}

func (c *Connection) Reconnect() {
	WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed. Ignore its error, as we're
	// redialing anyway.
	c.Close()

	c.mut.Lock()
	c.reconnection = true
	c.mut.Unlock()

	for i := 1; ; i++ {
		WSDebug("Trying to dial, attempt #", i)

		// Condition: err == ErrInvalidSession:
		// If the connection is rate limited (documented behavior):
		// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

		if err := c.Open(); err != nil {
			c.ErrorLog(errors.Wrap(err, "Failed to open gateway"))
			continue
		}

		WSDebug("Started after attempt: ", i)
		return
	}
}

func (c *Connection) Disconnect(g *gateway.Gateway) (err error) {
	if c.SessionID != "" {
		err = g.UpdateVoiceState(gateway.UpdateVoiceStateData{
			GuildID:   c.GuildID,
			ChannelID: nil,
			SelfMute:  true,
			SelfDeaf:  true,
		})

		c.SessionID = ""
	}

	// We might want this error and the update voice state error
	// but for now we will prioritize the voice state error
	_ = c.Close()

	return
}

// handleWS .
func (c *Connection) handleWS() {
	err := c.eventLoop()
	c.waitGroup.Done() // mark so Close() can exit.
	WSDebug("Event loop stopped.")

	if err != nil {
		c.ErrorLog(err)
		c.Reconnect()
		// Reconnect should spawn another eventLoop in its Start function.
	}
}

// eventLoop .
func (c *Connection) eventLoop() error {
	ch := c.WS.Listen()

	for {
		select {
		case err := <-c.paceDeath:
			// Got a paceDeath, we're exiting from here on out.
			c.paceDeath = nil // mark

			if err == nil {
				WSDebug("Pacemaker stopped without errors.")
				// No error, just exit normally.
				return nil
			}

			return errors.Wrap(err, "Pacemaker died, reconnecting")

		case ev := <-ch:
			// Handle the event
			if err := HandleEvent(c, ev); err != nil {
				c.ErrorLog(errors.Wrap(err, "WS handler error"))
			}
		}
	}
}

// Send .
func (c *Connection) Send(code OPCode, v interface{}) error {
	return c.send(code, v)
}

// send .
func (c *Connection) send(code OPCode, v interface{}) error {
	if c.WS == nil {
		return errors.New("tried to send data to a connection without a Websocket")
	}

	if c.WS.Conn == nil {
		return errors.New("tried to send data to a connection with a closed Websocket")
	}

	var op = OP{
		Code: code,
	}

	if v != nil {
		b, err := c.Driver.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "Failed to encode v")
		}

		op.Data = b
	}

	b, err := c.Driver.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "Failed to encode payload")
	}

	// WS should already be thread-safe.
	return c.WS.Send(b)
}
