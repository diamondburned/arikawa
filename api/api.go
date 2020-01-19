// Package api provides an interface to interact with the Discord REST API. It
// handles rate limiting, as well as authorizing and more.
package api

import (
	"net/http"

	"github.com/diamondburned/arikawa/api/rate"
	"github.com/diamondburned/arikawa/internal/httputil"
)

const (
	BaseEndpoint = "https://discordapp.com/api"
	APIVersion   = "6"

	Endpoint           = BaseEndpoint + "/v" + APIVersion + "/"
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
		Limiter: rate.NewLimiter(),
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
	tw.Post = func(r *http.Response) error {
		return cli.Limiter.Release(r.Request.URL.Path, r.Header)
	}

	cli.Client.Transport = tw

	return cli
}
