package gateway

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/httputil"
	"github.com/diamondburned/arikawa/json"
	"github.com/diamondburned/arikawa/wsutil"
	"github.com/pkg/errors"
)

const (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = "6"
	Encoding = "json"
)

var WSTimeout = wsutil.DefaultTimeout

func Gateway() (string, error) {
	var Gateway struct {
		URL string `json:"url"`
	}

	return Gateway.URL, httputil.DefaultClient.RequestJSON(
		&Gateway, "GET", EndpointGateway)
}

type Conn struct {
	ws *wsutil.Websocket
	json.Driver

	events chan interface{}

	Gateway     string // URL
	gatewayOnce sync.Once

	ErrorLog func(err error) // default to log.Println

	// Heartrate is the received duration between heartbeats.
	Heartrate time.Duration
	hrMutex   sync.Mutex

	// LastBeat logs the received heartbeats, with the newest one
	// first.
	LastBeat [2]time.Time

	// Used for Close()
	stoppers []chan<- struct{}
	closers  []func() error
}

func NewConn(driver json.Driver) (*Conn, error) {
	g, err := Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get gateway endpoint")
	}

	c := &Conn{
		Gateway: g,
		Driver:  driver,
	}

	param := url.Values{}
	param.Set("v", Version)
	param.Set("encoding", Encoding)

	ctx, cancel := context.WithTimeout(context.Background(), WSTimeout)
	defer cancel()

	ws, err := wsutil.New(ctx, driver, g)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Gateway "+g)
	}

	c.ws = ws
	return c, nil
}

func (c *Conn) Send(code OPCode, v interface{}) error {
	var op = OP{
		Code: code,
	}

	if v != nil {
		b, err := c.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "Failed to encode v")
		}

		op.Data = b
	}

	b, err := c.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "Failed to encode payload")
	}

	return c.ws.Send(b)
}

func (c *Conn) Close() error {
	for _, stop := range c.stoppers {
		close(stop)
	}

	var err error

	for _, closer := range c.closers {
		if cerr := closer(); cerr != nil {
			err = cerr
		}
	}

	return err
}

// StartGateway is called by New and should only be called once. This method is
// guarded with a sync.Do.
func (c *Conn) StartGateway() (chan Event, error) {
	var err error
	c.gatewayOnce.Do(func() {
		err = c.startGateway()
	})
	return err
}

// Reconnects and resumes.
func (c *Conn) Reconnect() error {
	panic("TODO")
}

func (c *Conn) startGateway() error {
	// This is where we'll get our events
	ch := c.ws.Listen()

	// Wait for an OP 10 Hello
	var hello HelloEvent
	if err := AssertEvent(c, <-ch, HeartbeatOP, &hello); err != nil {
		return errors.Wrap(err, "Error at Hello")
	}

	// Start the pacemaker with the heartrate received from Hello
	c.Heartrate = hello.HeartbeatInterval.Duration()
	go c.startPacemaker()

	return nil
}

func (c *Conn) startListener() {
	ch := c.ws.Listen()
	stop := c.stopper()

	for {
		select {
		case <-stop:
			return
		case v, ok := <-ch:
			if !ok {
				return
			}

			op, err := DecodeOP(c, v)
			if err != nil {
				c.ErrorLog(errors.Wrap(err, "Failed to decode OP in loop"))
			}

			if err := c.handleOP(op); err != nil {
				c.ErrorLog(err)
			}
		}
	}
}

func (c *Conn) handleOP(op *OP) error {
	switch op.Code {
	case HeartbeatAckOP:
		// Swap our received heartbeats
		c.LastBeat[0], c.LastBeat[1] = time.Now(), c.LastBeat[0]
	}

	return nil
}

func (c *Conn) startPacemaker() {
	stop := c.stopper()
	tick := time.NewTicker(c.Heartrate)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			if err := c.Heartbeat(); err != nil {
				c.ErrorLog(errors.Wrap(err, "Failed to send heartbeat"))
			}

			// Check and see if heartbeats have timed out.
			// TODO: better way?
			if c.LastBeat[0].Sub(c.LastBeat[1]) > c.Heartrate {
				if err := c.Reconnect(); err != nil {
					c.ErrorLog(errors.Wrap(err,
						"Failed to reconnect after heartrate timeout"))
				}
			}
		}
	}
}

func (c *Conn) stopper() <-chan struct{} {
	stop := make(chan struct{})
	c.stoppers = append(c.stoppers, stop)
	return stop
}
