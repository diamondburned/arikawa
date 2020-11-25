// Package httputil provides abstractions around the common needs of HTTP. It
// also allows swapping in and out the HTTP client.
package httputil

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v2/utils/json"
)

// StatusTooManyRequests is the HTTP status code discord sends on rate-limiting.
const StatusTooManyRequests = 429

// Retries is the default attempts to retry if the API returns an error before
// giving up. If the value is smaller than 1, then requests will retry forever.
var Retries uint = 5

type Client struct {
	httpdriver.Client
	SchemaEncoder

	// OnRequest, if not nil, will be copied and prefixed on each Request.
	OnRequest []RequestOption

	// OnResponse is called after every Do() call. Response might be nil if Do()
	// errors out. The error returned will override Do's if it's not nil.
	OnResponse []ResponseFunc

	// Timeout is the maximum amount of time the client will wait for a request
	// to finish. If this is 0 or smaller the Client won't time out. Otherwise,
	// the timeout will be used as deadline for context of every request.
	Timeout time.Duration

	// Default to the global Retries variable (5).
	Retries uint

	context context.Context
}

func NewClient() *Client {
	return &Client{
		Client:        httpdriver.NewClient(),
		SchemaEncoder: &DefaultSchema{},
		Retries:       Retries,
		context:       context.Background(),
	}
}

// Copy returns a shallow copy of the client.
func (c *Client) Copy() *Client {
	cl := new(Client)
	*cl = *c
	return cl
}

// WithContext returns a client copy of the client with the given context.
func (c *Client) WithContext(ctx context.Context) *Client {
	c = c.Copy()
	c.context = ctx
	return c
}

// Context is a shared context for all future calls. It's Background by
// default.
func (c *Client) Context() context.Context {
	return c.context
}

// applyOptions tries to apply all options. It does not halt if a single option
// fails, and the error returned is the latest error.
func (c *Client) applyOptions(r httpdriver.Request, extra []RequestOption) (e error) {
	for _, opt := range c.OnRequest {
		if err := opt(r); err != nil {
			e = err
		}
	}

	for _, opt := range extra {
		if err := opt(r); err != nil {
			e = err
		}
	}

	return
}

func (c *Client) MeanwhileMultipart(
	writer func(*multipart.Writer) error,
	method, url string, opts ...RequestOption) (httpdriver.Response, error) {

	r, w := io.Pipe()
	body := multipart.NewWriter(w)

	go func() { w.CloseWithError(writer(body)) }()

	// Prepend the multipart writer and the correct Content-Type header options.
	opts = PrependOptions(
		opts,
		WithBody(r),
		WithContentType(body.FormDataContentType()),
	)

	// Request with the current client and our own context:
	return c.Request(method, url, opts...)
}

func (c *Client) FastRequest(method, url string, opts ...RequestOption) error {
	r, err := c.Request(method, url, opts...)
	if err != nil {
		return err
	}

	return r.GetBody().Close()
}

func (c *Client) RequestJSON(to interface{}, method, url string, opts ...RequestOption) error {
	opts = PrependOptions(opts, JSONRequest)

	r, err := c.Request(method, url, opts...)
	if err != nil {
		return err
	}

	var body, status = r.GetBody(), r.GetStatus()
	defer body.Close()

	// No content, working as intended (tm)
	if status == httpdriver.NoContent {
		return nil
	}

	if err := json.DecodeStream(body, to); err != nil {
		return JSONError{err}
	}

	return nil
}

func (c *Client) Request(method, url string, opts ...RequestOption) (httpdriver.Response, error) {
	var doErr error

	var r httpdriver.Response
	var status int

	ctx := c.context

	if c.Timeout > 0 {
		var cancel func()

		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	// The c.Retries < 1 check ensures that we retry forever if that field is
	// less than 1.
	for i := uint(0); c.Retries < 1 || i < c.Retries; i++ {
		q, err := c.Client.NewRequest(ctx, method, url)
		if err != nil {
			return nil, RequestError{err}
		}

		if err := c.applyOptions(q, opts); err != nil {
			// We failed to apply an option, so we should call all OnResponse
			// handler to clean everything up.
			for _, fn := range c.OnResponse {
				fn(q, nil)
			}
			// Exit after cleaning everything up.
			return nil, errors.Wrap(err, "failed to apply http request options")
		}

		r, doErr = c.Client.Do(q)

		// Error that represents the latest error in the chain.
		var onRespErr error

		// Call OnResponse() even if the request failed.
		for _, fn := range c.OnResponse {
			// Be sure to call ALL OnResponse handlers.
			if err := fn(q, r); err != nil {
				onRespErr = err
			}
		}

		if onRespErr != nil {
			return nil, errors.Wrap(err, "OnResponse handler failed")
		}

		// Retry if the request failed.
		if doErr != nil {
			continue
		}

		if status = r.GetStatus(); status == StatusTooManyRequests || status >= 500 {
			continue
		}

		break
	}

	// If all retries failed:
	if doErr != nil {
		return nil, RequestError{doErr}
	}

	// Response received, but with a failure status code:
	if status < 200 || status > 299 {
		// Try and parse the body.
		var body = r.GetBody()
		defer body.Close()

		// This rarely happens, so we can (probably) make an exception for it.
		buf := bytes.Buffer{}
		buf.ReadFrom(body)

		httpErr := &HTTPError{
			Status: status,
			Body:   buf.Bytes(),
		}

		// Optionally unmarshal the error.
		json.Unmarshal(httpErr.Body, &httpErr)

		return nil, httpErr
	}

	return r, nil
}
