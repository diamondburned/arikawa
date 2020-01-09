package gateway

import (
	"net/url"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/httputil"
	"github.com/pkg/errors"
)

const (
	EndpointGateway    = api.Endpoint + "gateway"
	EndpointGatewayBot = api.EndpointGateway + "/bot"

	Version  = "6"
	Encoding = "json"
)

func Gateway() (string, error) {
	var Gateway struct {
		URL string `json:"url"`
	}

	return Gateway.URL, httputil.DefaultClient.RequestJSON(
		&Gateway, "GET", EndpointGateway)
}

type Conn struct {
	Gateway string // URL
	Token   string
}

func NewConn(token string) (*Conn, error) {
	g, err := Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get gateway endpoint")
	}

	c := &Conn{
		Gateway: g,
		Token:   token,
	}

	param := url.Values{}
	param.Set("v", Version)
	param.Set("encoding", Encoding)

	return c, nil
}
