// Package gateway handles the Discord gateway (or Websocket) connection, its
// events, and everything related to it. This includes logging into the
// Websocket.
//
// This package does not abstract events and function handlers; instead, it
// leaves that to the session package. This package exposes only a single Events
// channel.
package gateway

import (
	"context"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/internal/httputil"
	"github.com/diamondburned/arikawa/internal/json"
	"github.com/diamondburned/arikawa/internal/wsutil"
	"github.com/pkg/errors"
)

const (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = "6"
	Encoding = "json"
	// Compress = "zlib-stream"
)

var (
	// WSTimeout is the timeout for connecting and writing to the Websocket,
	// before Gateway cancels and fails.
	WSTimeout = wsutil.DefaultTimeout
	// WSBuffer is the size of the Event channel. This has to be at least 1 to
	// make space for the first Event: Ready or Resumed.
	WSBuffer = 10
	// WSError is the default error handler
	WSError = func(err error) { log.Println("Gateway error:", err) }
	// WSExtraReadTimeout is the duration to be added to Hello, as a read
	// timeout for the websocket.
	WSExtraReadTimeout = time.Second
	// WSRetries controls the number of Reconnects before erroring out.
	WSRetries = 3
	// WSDebug is used for extra debug logging. This is expected to behave
	// similarly to log.Println().
	WSDebug = func(v ...interface{}) {}
)

var (
	ErrMissingForResume = errors.New(
		"missing session ID or sequence for resuming")
	ErrWSMaxTries = errors.New("max tries reached")
)

func GatewayURL() (string, error) {
	var Gateway struct {
		URL string `json:"url"`
	}

	return Gateway.URL, httputil.DefaultClient.RequestJSON(
		&Gateway, "GET", EndpointGateway)
}

type Gateway struct {
	WS *wsutil.Websocket
	json.Driver

	// Timeout for connecting and writing to the Websocket, uses default
	// WSTimeout (global).
	WSTimeout time.Duration

	// All events sent over are pointers to Event structs (structs suffixed with
	// "Event"). This shouldn't be accessed if the Gateway is created with a
	// Session.
	Events chan Event

	SessionID string

	Identifier *Identifier
	Pacemaker  *Pacemaker
	Sequence   *Sequence

	ErrorLog func(err error) // default to log.Println

	// AfterClose is called after each close. Error can be non-nil, as this is
	// called even when the Gateway is gracefully closed. It's used mainly for
	// reconnections or any type of connection interruptions.
	AfterClose func(err error) // noop by default

	// FatalError is where Reconnect errors will go to. When an error is sent
	// here, the Gateway is already dead, so Close() shouldn't be called.
	// This channel is buffered once.
	FatalError <-chan error
	fatalError chan error

	// Only use for debugging

	// If this channel is non-nil, all incoming OP packets will also be sent
	// here. This should be buffered, so to not block the main loop.
	OP chan *OP

	// Mutex to hold off calls when the WS is not available. Doesn't block if
	// Start() is not called or Close() is called. Also doesn't block for
	// Identify or Resume.
	// available sync.RWMutex

	// Filled by methods, internal use
	paceDeath chan error
	waitGroup *sync.WaitGroup
}

// NewGateway starts a new Gateway with the default stdlib JSON driver. For more
// information, refer to NewGatewayWithDriver.
func NewGateway(token string) (*Gateway, error) {
	return NewGatewayWithDriver(token, json.Default{})
}

// NewGatewayWithDriver connects to the Gateway and authenticates automatically.
func NewGatewayWithDriver(token string, driver json.Driver) (*Gateway, error) {
	URL, err := GatewayURL()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get gateway endpoint")
	}

	g := &Gateway{
		Driver:     driver,
		WSTimeout:  WSTimeout,
		Events:     make(chan Event, WSBuffer),
		Identifier: DefaultIdentifier(token),
		Sequence:   NewSequence(),
		ErrorLog:   WSError,
		AfterClose: func(error) {},
		fatalError: make(chan error, 1),
	}
	g.FatalError = g.fatalError

	// Parameters for the gateway
	param := url.Values{
		"v":        {Version},
		"encoding": {Encoding},
	}

	// Append the form to the URL
	URL += "?" + param.Encode()

	// Create a new undialed Websocket.
	g.WS = wsutil.NewCustom(wsutil.NewConn(driver), URL)

	// Try and dial it
	return g, nil
}

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() error {
	// Check if the WS is already closed:
	if g.waitGroup == nil && g.paceDeath == nil {
		WSDebug("Gateway is already closed.")

		g.AfterClose(nil)
		return nil
	}

	// If the pacemaker is running:
	if g.paceDeath != nil {
		WSDebug("Stopping pacemaker...")

		// Stop the pacemaker and the event handler
		g.Pacemaker.Stop()

		WSDebug("Stopped pacemaker.")
	}

	WSDebug("Waiting for WaitGroup to be done.")

	// This should work, since Pacemaker should signal its loop to stop, which
	// would also exit our event loop. Both would be 2.
	g.waitGroup.Wait()

	WSDebug("WaitGroup is done.")

	// Mark g.waitGroup as empty:
	g.waitGroup = nil

	// Stop the Websocket
	err := g.WS.Close()
	g.AfterClose(err)
	return err
}

// Reconnects and resumes.
func (g *Gateway) Reconnect() error {
	WSDebug("Reconnecting...")

	// Guarantee the gateway is already closed:
	if err := g.Close(); err != nil {
		return errors.Wrap(err, "Failed to close Gateway before reconnecting")
	}

	for i := 0; i < WSRetries; i++ {
		WSDebug("Trying to dial, attempt", i)

		// Condition: err == ErrInvalidSession:
		// If the connection is rate limited (documented behavior):
		// https://discordapp.com/developers/docs/topics/gateway#rate-limiting

		if err := g.Open(); err != nil && err != ErrInvalidSession {
			g.ErrorLog(errors.Wrap(err, "Failed to open gateway"))
			continue
		}

		WSDebug("Started after attempt:", i)
		return nil
	}

	return ErrWSMaxTries
}

// Open connects to the Websocket and authenticate it. You should usually use
// this function over Start().
func (g *Gateway) Open() error {
	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	// Reconnect to the Gateway
	if err := g.WS.Dial(ctx); err != nil {
		return errors.Wrap(err, "Failed to reconnect")
	}

	WSDebug("Trying to start...")

	// Try to resume the connection
	if err := g.Start(); err != nil {
		return err
	}

	// Started successfully, return
	return nil
}

// Start authenticates with the websocket, or resume from a dead Websocket
// connection. This function doesn't block. You wouldn't usually use this
// function, but Open() instead.
func (g *Gateway) Start() error {
	// g.available.Lock()
	// defer g.available.Unlock()

	if err := g.start(); err != nil {
		WSDebug("Start failed:", err)

		// Close can be called with the mutex still acquired here, as the
		// pacemaker hasn't started yet.
		if err := g.Close(); err != nil {
			WSDebug("Failed to close after start fail:", err)
		}
		return err
	}
	return nil
}

// Wait blocks until the Gateway fatally exits when it couldn't reconnect
// anymore. To use this withh other channels, check out g.FatalError. If a
// non-nil error is returned, Close() shouldn't be called again.
func (g *Gateway) Wait() error {
	return <-g.FatalError
}

func (g *Gateway) start() error {
	// This is where we'll get our events
	ch := g.WS.Listen()

	// Make a new WaitGroup for use in background loops:
	g.waitGroup = new(sync.WaitGroup)

	// Wait for an OP 10 Hello
	var hello HelloEvent
	if _, err := AssertEvent(g, <-ch, HelloOP, &hello); err != nil {
		return errors.Wrap(err, "Error at Hello")
	}

	// Send Discord either the Identify packet (if it's a fresh connection), or
	// a Resume packet (if it's a dead connection).
	if g.SessionID == "" {
		// SessionID is empty, so this is a completely new session.
		if err := g.Identify(); err != nil {
			return errors.Wrap(err, "Failed to identify")
		}
	} else {
		if err := g.Resume(); err != nil {
			return errors.Wrap(err, "Failed to resume")
		}
	}

	// Expect either READY or RESUMED before continuing.
	WSDebug("Waiting for either READY or RESUMED.")

	err := WaitForEvent(g, ch, func(op *OP) bool {
		switch op.EventName {
		case "READY":
			WSDebug("Found READY event.")
			return true
		case "RESUMED":
			WSDebug("Found RESUMED event.")
			return true
		}
		return false
	})

	if err != nil {
		return errors.Wrap(err, "First error")
	}

	// Start the pacemaker with the heartrate received from Hello, after
	// initializing everything. This ensures we only heartbeat if the websocket
	// is authenticated.
	g.Pacemaker = &Pacemaker{
		Heartrate: hello.HeartbeatInterval.Duration(),
		Pace:      g.Heartbeat,
		OnDead:    g.Reconnect,
	}
	// Pacemaker dies here, only when it's fatal.
	g.paceDeath = g.Pacemaker.StartAsync(g.waitGroup)

	// Start the event handler
	g.waitGroup.Add(1)
	go g.handleWS()

	WSDebug("Started successfully.")

	return nil
}

// handleWS uses the Websocket and parses them into g.Events.
func (g *Gateway) handleWS() {
	err := g.eventLoop()
	g.waitGroup.Done()

	if err != nil {
		g.ErrorLog(err)

		g.fatalError <- errors.Wrap(g.Reconnect(), "Failed to reconnect")
		// Reconnect should spawn another eventLoop in its Start function.
	}
}

func (g *Gateway) eventLoop() error {
	ch := g.WS.Listen()

	for {
		select {
		case err := <-g.paceDeath:
			// Got a paceDeath, we're exiting from here on out.
			g.paceDeath = nil // mark

			if err == nil {
				WSDebug("Pacemaker stopped without errors.")
				// No error, just exit normally.
				return nil
			}

			return errors.Wrap(err, "Pacemaker died, reconnecting")

		case ev := <-ch:
			// Handle the event
			if err := HandleEvent(g, ev); err != nil {
				g.ErrorLog(errors.Wrap(err, "WS handler error"))
			}
		}
	}
}

func (g *Gateway) Send(code OPCode, v interface{}) error {
	return g.send(true, code, v)
}

func (g *Gateway) send(lock bool, code OPCode, v interface{}) error {
	var op = OP{
		Code: code,
	}

	if v != nil {
		b, err := g.Driver.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "Failed to encode v")
		}

		op.Data = b
	}

	b, err := g.Driver.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "Failed to encode payload")
	}

	// if lock {
	// 	g.available.RLock()
	// 	defer g.available.RUnlock()
	// }

	// WS should already be thread-safe.
	return g.WS.Send(b)
}
