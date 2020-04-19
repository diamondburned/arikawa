// Package api provides an interface to interact with the Discord REST API. It
// handles rate limiting, as well as authorizing and more.
package api

import (
	"net/http"

	"github.com/diamondburned/arikawa/api/rate"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
)

var (
	BaseEndpoint = "https://discordapp.com"
	APIVersion   = "6"
	APIPath      = "/api/v" + APIVersion

	Endpoint           = BaseEndpoint + APIPath + "/"
	EndpointGateway    = Endpoint + "gateway"
	EndpointGatewayBot = EndpointGateway + "/bot"
)

var UserAgent = "DiscordBot (https://github.com/diamondburned/arikawa, v0.0.1)"

type Client struct {
	*httputil.Client
	Limiter *rate.Limiter

	Token     string
	UserAgent string
}

func NewClient(token string) *Client {
	return NewCustomClient(token, httputil.NewClient())
}

func NewCustomClient(token string, httpClient *httputil.Client) *Client {
	cli := &Client{
		Client:    httpClient,
		Limiter:   rate.NewLimiter(APIPath),
		Token:     token,
		UserAgent: UserAgent,
	}

	cli.DefaultOptions = []httputil.RequestOption{
		func(r httpdriver.Request) error {
			r.AddHeader(http.Header{
				"Authorization":         {cli.Token},
				"User-Agent":            {cli.UserAgent},
				"X-RateLimit-Precision": {"millisecond"},
			})

			// Rate limit stuff
			return cli.Limiter.Acquire(r.GetContext(), r.GetPath())
		},
	}
	cli.OnResponse = func(r httpdriver.Request, resp httpdriver.Response) error {
		return cli.Limiter.Release(r.GetPath(), httpdriver.OptHeader(resp))
	}

	return cli
}
