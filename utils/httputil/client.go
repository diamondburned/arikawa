// Package httputil provides abstractions around the common needs of HTTP. It
// also allows swapping in and out the HTTP client.
package httputil

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"

	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/pkg/errors"
)

// Retries is the default attempts to retry if the API returns an error before
// giving up. If the value is smaller than 1, then requests will retry forever.
var Retries uint = 5

type Client struct {
	httpdriver.Client
	json.Driver
	SchemaEncoder

	// DefaultOptions, if not nil, will be copied and prefixed on each Request.
	DefaultOptions []RequestOption

	// OnResponse is called after every Do() call. Response might be nil if Do()
	// errors out. The error returned will override Do's if it's not nil.
	OnResponse func(httpdriver.Request, httpdriver.Response) error

	// Default to the global Retries variable (5).
	Retries uint
}

// ResponseNoop is used for (*Client).OnResponse.
func ResponseNoop(httpdriver.Request, httpdriver.Response) error {
	return nil
}

func NewClient() *Client {
	return &Client{
		Client:        httpdriver.NewClient(),
		Driver:        json.Default,
		SchemaEncoder: &DefaultSchema{},
		Retries:       Retries,
		OnResponse:    ResponseNoop,
	}
}

func (c *Client) applyOptions(r httpdriver.Request, extra []RequestOption) error {
	for _, opt := range c.DefaultOptions {
		if err := opt(r); err != nil {
			return err
		}
	}
	for _, opt := range extra {
		if err := opt(r); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) MeanwhileMultipart(
	writer func(*multipart.Writer) error,
	method, url string, opts ...RequestOption) (httpdriver.Response, error) {

	// We want to cancel the request if our bodyWriter fails
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, w := io.Pipe()
	body := multipart.NewWriter(w)

	var bgErr error

	go func() {
		if err := writer(body); err != nil {
			bgErr = err
			cancel()
		}

		// Close the writer so the body gets flushed to the HTTP reader.
		w.Close()
	}()

	// Prepend the multipart writer and the correct Content-Type header options.
	opts = PrependOptions(
		opts,
		WithBody(r),
		WithContentType(body.FormDataContentType()),
	)

	resp, err := c.RequestCtx(ctx, method, url, opts...)
	if err != nil && bgErr != nil {
		return nil, bgErr
	}
	return resp, err
}

func (c *Client) FastRequest(method, url string, opts ...RequestOption) error {
	r, err := c.Request(method, url, opts...)
	if err != nil {
		return err
	}

	return r.GetBody().Close()
}

func (c *Client) RequestCtxJSON(
	ctx context.Context,
	to interface{}, method, url string, opts ...RequestOption) error {

	opts = PrependOptions(opts, JSONRequest)

	r, err := c.RequestCtx(ctx, method, url, opts...)
	if err != nil {
		return err
	}

	var body, status = r.GetBody(), r.GetStatus()
	defer body.Close()

	// No content, working as intended (tm)
	if status == httpdriver.NoContent {
		return nil
	}

	if err := c.DecodeStream(body, to); err != nil {
		return JSONError{err}
	}

	return nil
}

func (c *Client) RequestCtx(
	ctx context.Context,
	method, url string, opts ...RequestOption) (httpdriver.Response, error) {

	req, err := c.Client.NewRequest(ctx, method, url)
	if err != nil {
		return nil, RequestError{err}
	}

	if err := c.applyOptions(req, opts); err != nil {
		return nil, errors.Wrap(err, "Failed to apply options")
	}

	var r httpdriver.Response
	var status int

	for i := uint(0); c.Retries < 1 || i < c.Retries; i++ {
		r, err = c.Client.Do(req)
		if err != nil {
			continue
		}

		if status = r.GetStatus(); status < 200 || status > 299 {
			continue
		}

		break
	}

	// Call OnResponse() even if the request failed.
	if err := c.OnResponse(req, r); err != nil {
		return nil, err
	}

	// If all retries failed:
	if err != nil {
		return nil, RequestError{err}
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
		c.Unmarshal(httpErr.Body, &httpErr)

		return nil, httpErr
	}

	return r, nil
}

func (c *Client) Request(method, url string, opts ...RequestOption) (httpdriver.Response, error) {
	return c.RequestCtx(context.Background(), method, url, opts...)
}

func (c *Client) RequestJSON(to interface{}, method, url string, opts ...RequestOption) error {
	return c.RequestCtxJSON(context.Background(), to, method, url, opts...)
}
