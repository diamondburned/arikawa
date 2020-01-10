package gateway

import (
	"context"
	"net/url"

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
	*wsutil.Websocket
	JSON json.Driver

	Gateway string // URL
}

func NewConn(driver json.Driver) (*Conn, error) {
	g, err := Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get gateway endpoint")
	}

	c := &Conn{
		Gateway: g,
		JSON:    driver,
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

	c.Websocket = ws
	return c, nil
}

func (c *Conn) Send(code OPCode, v interface{}) error {
	var op = OP{
		Code: code,
	}

	if v != nil {
		b, err := c.JSON.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "Failed to encode v")
		}

		op.Data = b
	}

	b, err := c.JSON.Marshal(op)
	if err != nil {
		return errors.Wrap(err, "Failed to encode payload")
	}

	return c.Websocket.Send(b)
}

func (c *Conn) Heartbeat() error {
	return c.Send(HeartbeatOP, nil)
}

func (c *Conn) Identify(d IdentifyData) error {
	return c.Send(IdentifyOP, d)
}
