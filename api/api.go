// Package api provides an interface to interact with the Discord REST API. It
// handles rate limiting, as well as authorizing and more.
package api

import (
	"context"
	"net/http"

	"github.com/diamondburned/arikawa/api/rate"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
)

var (
	BaseEndpoint = "https://discord.com"
	Version      = "6"
	Path         = "/api/v" + Version

	Endpoint           = BaseEndpoint + Path + "/"
	EndpointGateway    = Endpoint + "gateway"
	EndpointGatewayBot = EndpointGateway + "/bot"
)

var UserAgent = "DiscordBot (https://github.com/diamondburned/arikawa, v0.0.1)"

type Client struct {
	*httputil.Client
	Session
}

func NewClient(token string) *Client {
	return NewCustomClient(token, httputil.NewClient())
}

func NewCustomClient(token string, httpClient *httputil.Client) *Client {
	ses := Session{
		Limiter:   rate.NewLimiter(Path),
		Token:     token,
		UserAgent: UserAgent,
	}

	hcl := httpClient.Copy()
	hcl.OnRequest = append(hcl.OnRequest, ses.InjectRequest)
	hcl.OnResponse = append(hcl.OnResponse, ses.OnResponse)

	return &Client{
		Client:  hcl,
		Session: ses,
	}
}

// WithContext returns a shallow copy of Client with the given context. It's
// used for method timeouts and such. This method is thread-safe.
func (c *Client) WithContext(ctx context.Context) *Client {
	return &Client{
		Client:  c.Client.WithContext(ctx),
		Session: c.Session,
	}
}

// Session keeps a single session. This is typically wrapped around Client.
type Session struct {
	Limiter *rate.Limiter

	Token     string
	UserAgent string
}

func (s *Session) InjectRequest(r httpdriver.Request) error {
	r.AddHeader(http.Header{
		"Authorization":         {s.Token},
		"User-Agent":            {s.UserAgent},
		"X-RateLimit-Precision": {"millisecond"},
	})

	// Rate limit stuff
	return s.Limiter.Acquire(r.GetContext(), r.GetPath())
}

func (s *Session) OnResponse(r httpdriver.Request, resp httpdriver.Response) error {
	return s.Limiter.Release(r.GetPath(), httpdriver.OptHeader(resp))
}
