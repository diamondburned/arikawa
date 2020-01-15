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
	"runtime"
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
)

var (
	// WSTimeout is the timeout for connecting and writing to the Websocket,
	// before Gateway cancels and fails.
	WSTimeout = wsutil.DefaultTimeout
	// WSBuffer is the size of the Event channel. This has to be at least 1 to
	// make space for the first Event: Ready or Resumed.
	WSBuffer = 10
	// WSRetries is the times Gateway would try and connect or reconnect to the
	// gateway.
	WSRetries = uint(5)
	// WSError is the default error handler
	WSError = func(err error) {}
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

// Identity is used as the default identity when initializing a new Gateway.
var Identity = IdentifyProperties{
	OS:      runtime.GOOS,
	Browser: "Arikawa",
	Device:  "Arikawa",
}

type Gateway struct {
	WS *wsutil.Websocket
	json.Driver

	// Timeout for connecting and writing to the Websocket, uses default
	// WSTimeout (global).
	WSTimeout time.Duration
	// Retries on connect and reconnect.
	WSRetries uint // 3

	// All events sent over are pointers to Event structs (structs suffixed with
	// "Event")
	Events chan Event

	SessionID string

	Identifier *Identifier
	Pacemaker  *Pacemaker
	Sequence   *Sequence

	ErrorLog func(err error) // default to log.Println

	// Only use for debugging

	// If this channel is non-nil, all incoming OP packets will also be sent
	// here. This should be buffered, so to not block the main loop.
	OP chan Event

	// Filled by methods, internal use
	paceDeath <-chan error
	handler   chan struct{}
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
		WSRetries:  WSRetries,
		Events:     make(chan Event, WSBuffer),
		Identifier: DefaultIdentifier(token),
		Sequence:   NewSequence(),
		ErrorLog:   WSError,
	}

	// Parameters for the gateway
	param := url.Values{}
	param.Set("v", Version)
	param.Set("encoding", Encoding)
	// Append the form to the URL
	URL += "?" + param.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	// Create a new undialed Websocket.
	ws, err := wsutil.NewCustom(ctx, wsutil.NewConn(driver), URL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Gateway "+URL)
	}
	g.WS = ws

	// Try and dial it
	return g, g.connect()
}

// Close closes the underlying Websocket connection.
func (g *Gateway) Close() error {
	log.Println("Stopping pacemaker")
	// Stop the pacemaker
	g.Pacemaker.Stop()

	// Stop the Websocket
	return g.WS.Close(nil)
}

// Reconnects and resumes.
func (g *Gateway) Reconnect() error {
	// Close, but we don't care about the error (I think)
	g.Close()
	// Actually a reconnect at this point.
	return g.connect()
}

// Resume sends to the Websocket a Resume OP, but it doesn't actually resume
// from a dead connection. Start() resumes from a dead connection.
func (g *Gateway) Resume() error {
	var (
		ses = g.SessionID
		seq = g.Sequence.Get()
	)

	if ses == "" || seq == 0 {
		return ErrMissingForResume
	}

	return g.Send(ResumeOP, ResumeData{
		Token:     g.Identifier.Token,
		SessionID: ses,
		Sequence:  seq,
	})
}

// Start authenticates with the websocket, or resume from a dead Websocket
// connection. This function doesn't block. To block until a fatal error, use
// Wait().
func (g *Gateway) Start() error {
	// This is where we'll get our events
	ch := g.WS.Listen()

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

		// We should now expect a Ready event.
		var ready ReadyEvent
		p, err := AssertEvent(g, <-ch, DispatchOP, &ready)
		if err != nil {
			return errors.Wrap(err, "Error at Ready")
		}

		// We now also have the SessionID and the SequenceID
		g.SessionID = ready.SessionID
		g.Sequence.Set(p.Sequence)

		// Send the event away
		g.Events <- &ready

	} else {
		if err := g.Resume(); err != nil {
			return errors.Wrap(err, "Failed to resume")
		}

		// We should now expect a Resumed event.
		var resumed ResumedEvent
		_, err := AssertEvent(g, <-ch, DispatchOP, &resumed)
		if err != nil {
			return errors.Wrap(err, "Error at Resumed")
		}

		// Send the event away
		g.Events <- &resumed
	}

	// Start the event handler
	g.handler = make(chan struct{})
	go g.handleWS(g.handler)

	// Start the pacemaker with the heartrate received from Hello
	g.Pacemaker = &Pacemaker{
		Heartrate: hello.HeartbeatInterval.Duration(),
		Pace:      g.Heartbeat,
		OnDead:    g.Reconnect,
	}
	// Pacemaker dies here, only when it's fatal.
	g.paceDeath = g.Pacemaker.StartAsync()

	return nil
}

func (g *Gateway) Wait() error {
	defer close(g.handler)
	return <-g.paceDeath
}

// handleWS uses the Websocket and parses them into g.Events.
func (g *Gateway) handleWS(stop <-chan struct{}) {
	ch := g.WS.Listen()

	for {
		select {
		case <-stop:
			return
		case ev := <-ch:
			// Check for error
			if ev.Error != nil {
				g.ErrorLog(ev.Error)
				continue
			}

			// Handle the event
			if err := HandleEvent(g, ev.Data); err != nil {
				g.ErrorLog(errors.Wrap(ev.Error, "WS handler error"))
			}
		}
	}
}

func (g *Gateway) Send(code OPCode, v interface{}) error {
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

	log.Println("->", string(b))

	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.WS.Send(ctx, b)
}

func (g *Gateway) connect() error {
	// Reconnect timeout
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	var Lerr error

	for i := uint(0); i < g.WSRetries; i++ {
		// Check if context is expired
		if err := ctx.Err(); err != nil {
			// Don't bother if it's expired
			return err
		}

		// Reconnect to the Gateway
		if err := g.WS.Redial(ctx); err != nil {
			// Save the error, retry again
			Lerr = errors.Wrap(err, "Failed to reconnect")
			continue
		}

		// Try to resume the connection
		if err := g.Start(); err != nil {
			// If the connection is rate limited (documented behavior):
			// https://discordapp.com/developers/docs/topics/gateway#rate-limiting
			if err == ErrInvalidSession {
				continue // retry
			}

			// Else, fatal
			return errors.Wrap(err, "Failed to start gateway")
		}

		// Started successfully, return
		return nil
	}

	// Check if any earlier errors are fatal
	if Lerr != nil {
		return Lerr
	}

	// We tried.
	return ErrWSMaxTries
}
