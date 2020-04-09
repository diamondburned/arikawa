// Package httputil provides abstractions around the common needs of HTTP. It
// also allows swapping in and out the HTTP client.
package httputil

import (
	"context"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/utils/json"
)

// Retries is the default attempts to retry if the API returns an error before
// giving up.
var Retries uint = 5

type Client struct {
	http.Client
	json.Driver
	SchemaEncoder

	Retries uint
}

var DefaultClient = NewClient()

func NewClient() Client {
	return Client{
		Client: http.Client{
			Timeout: 10 * time.Second,
		},
		Driver:        json.Default{},
		SchemaEncoder: &DefaultSchema{},
		Retries:       Retries,
	}
}

func (c *Client) MeanwhileMultipart(
	multipartWriter func(*multipart.Writer) error,
	method, url string, opts ...RequestOption) (*http.Response, error) {

	// We want to cancel the request if our bodyWriter fails
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, w := io.Pipe()
	body := multipart.NewWriter(w)

	var bgErr error

	go func() {
		if err := multipartWriter(body); err != nil {
			bgErr = err
			cancel()
		}

		// Close the writer so the body gets flushed to the HTTP reader.
		w.Close()
	}()

	resp, err := c.RequestCtx(ctx, method, url,
		append([]RequestOption{
			WithBody(r),
			WithContentType(body.FormDataContentType()),
		}, opts...)...)

	if err != nil && bgErr != nil {
		if resp.Body != nil {
			resp.Body.Close()
		}

		return nil, bgErr
	}

	return resp, err
}

func (c *Client) FastRequest(
	method, url string, opts ...RequestOption) error {

	r, err := c.Request(method, url, opts...)
	if err != nil {
		return err
	}

	return r.Body.Close()
}

func (c *Client) RequestCtx(ctx context.Context,
	method, url string, opts ...RequestOption) (*http.Response, error) {

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, RequestError{err}
	}

	for _, opt := range opts {
		if err := opt(req); err != nil {
			return nil, err
		}
	}

	var r *http.Response

	for i := uint(0); i < c.Retries; i++ {
		r, err = c.Client.Do(req)
		if err != nil {
			continue
		}

		if r.StatusCode < 200 || r.StatusCode > 299 {
			continue
		}

		break
	}

	// If all retries failed:
	if err != nil {
		return nil, RequestError{err}
	}

	// Response received, but with a failure status code:
	if r.StatusCode < 200 || r.StatusCode > 299 {
		httpErr := &HTTPError{
			Status: r.StatusCode,
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, httpErr
		}

		httpErr.Body = b

		c.Unmarshal(b, &httpErr)
		return nil, httpErr
	}

	return r, nil
}

func (c *Client) RequestCtxJSON(ctx context.Context,
	to interface{}, method, url string, opts ...RequestOption) error {

	r, err := c.RequestCtx(ctx, method, url,
		append([]RequestOption{JSONRequest}, opts...)...)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	// No content, working as intended (tm)
	if r.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := c.DecodeStream(r.Body, to); err != nil {
		return JSONError{err}
	}

	return nil
}

func (c *Client) Request(
	method, url string, opts ...RequestOption) (*http.Response, error) {

	return c.RequestCtx(context.Background(), method, url, opts...)
}

func (c *Client) RequestJSON(
	to interface{}, method, url string, opts ...RequestOption) error {

	return c.RequestCtxJSON(context.Background(), to, method, url, opts...)
}
