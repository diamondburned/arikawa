// Package api provides an interface to interact with the Discord REST API. It
// handles rate limiting, as well as authorizing and more.
package api

import (
	"context"
	"net/http"

	"github.com/diamondburned/arikawa/v3/api/rate"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
)

var (
	BaseEndpoint = "https://discord.com"
	Version      = "9"
	Path         = "/api/v" + Version

	Endpoint           = BaseEndpoint + Path + "/"
	EndpointGateway    = Endpoint + "gateway"
	EndpointGatewayBot = EndpointGateway + "/bot"
)

var UserAgent = "DiscordBot (https://github.com/diamondburned/arikawa/v3)"

type Client struct {
	*httputil.Client
	*Session
	AcquireOptions rate.AcquireOptions
}

func NewClient(token string) *Client {
	return NewCustomClient(token, httputil.NewClient())
}

func NewCustomClient(token string, httpClient *httputil.Client) *Client {
	c := &Client{
		Session: &Session{
			Limiter:   rate.NewLimiter(Path),
			Token:     token,
			UserAgent: UserAgent,
		},
		Client: httpClient.Copy(),
	}

	c.Client.OnRequest = append(c.Client.OnRequest, c.InjectRequest)
	c.Client.OnResponse = append(c.Client.OnResponse, c.OnResponse)

	return c
}

// WithContext returns a shallow copy of Client with the given context. It's
// used for method timeouts and such. This method is thread-safe.
func (c *Client) WithContext(ctx context.Context) *Client {
	return &Client{
		Client:         c.Client.WithContext(ctx),
		Session:        c.Session,
		AcquireOptions: c.AcquireOptions,
	}
}

func (c *Client) InjectRequest(r httpdriver.Request) error {
	r.AddHeader(http.Header{
		"Authorization": {c.Session.Token},
		"User-Agent":    {c.Session.UserAgent},
	})

	ctx := c.AcquireOptions.Context(r.GetContext())
	return c.Session.Limiter.Acquire(ctx, r.GetPath())
}

func (c *Client) OnResponse(r httpdriver.Request, resp httpdriver.Response) error {
	return c.Session.Limiter.Release(r.GetPath(), httpdriver.OptHeader(resp))
}

// Session keeps a single session. This is typically wrapped around Client.
type Session struct {
	Limiter *rate.Limiter

	Token     string
	UserAgent string
}

// AuditLogReason is the type embedded in data structs when the action
// performed by calling that api endpoint supports attaching a custom audit log
// reason.
type AuditLogReason string

// Header returns a http.Header containing the reason, or nil if the reason is
// empty.
func (r AuditLogReason) Header() http.Header {
	if len(r) == 0 {
		return nil
	}

	return http.Header{"X-Audit-Log-Reason": []string{string(r)}}
}
