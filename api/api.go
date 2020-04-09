// Package api provides an interface to interact with the Discord REST API. It
// handles rate limiting, as well as authorizing and more.
package api

import (
	"net/http"

	"github.com/diamondburned/arikawa/api/rate"
	"github.com/diamondburned/arikawa/utils/httputil"
)

const (
	BaseEndpoint = "https://discordapp.com"
	APIVersion   = "6"
	APIPath      = "/api/v" + APIVersion

	Endpoint           = BaseEndpoint + APIPath + "/"
	EndpointGateway    = Endpoint + "gateway"
	EndpointGatewayBot = EndpointGateway + "/bot"
)

var UserAgent = "DiscordBot (https://github.com/diamondburned/arikawa, v0.0.1)"

type Client struct {
	httputil.Client
	Limiter *rate.Limiter

	Token string
}

func NewClient(token string) *Client {
	cli := &Client{
		Client:  httputil.DefaultClient,
		Limiter: rate.NewLimiter(APIPath),
		Token:   token,
	}

	tw := httputil.NewTransportWrapper()
	tw.Pre = func(r *http.Request) error {
		if cli.Token != "" {
			r.Header.Set("Authorization", cli.Token)
		}

		r.Header.Set("User-Agent", UserAgent)
		r.Header.Set("X-RateLimit-Precision", "millisecond")

		// Rate limit stuff
		return cli.Limiter.Acquire(r.Context(), r.URL.Path)
	}
	tw.Post = func(r *http.Request, resp *http.Response) error {
		if resp == nil {
			return cli.Limiter.Release(r.URL.Path, nil)
		}
		return cli.Limiter.Release(r.URL.Path, resp.Header)
	}

	cli.Client.Transport = tw

	return cli
}
