package httputil

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/json"
)

type Client struct {
	http.Client
	json.Driver
	SchemaEncoder
}

var DefaultClient = NewClient()

func NewClient() Client {
	return Client{
		Client: http.Client{
			Timeout: 10 * time.Second,
		},
		Driver:        json.Default{},
		SchemaEncoder: &DefaultSchema{},
	}
}

func (c *Client) MeanwhileBody(bodyWriter func(io.Writer) error,
	method, url string, opts ...RequestOption) (*http.Response, error) {

	// We want to cancel the request if our bodyWriter fails
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()

	var bgErr error

	go func() {
		if err := bodyWriter(w); err != nil {
			bgErr = err
			cancel()
		}
	}()

	resp, err := c.RequestCtx(ctx, method, url,
		append([]RequestOption{WithBody(r)}, opts...)...)

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

	r, err := c.Client.Do(req)
	if err != nil {
		return nil, RequestError{err}
	}

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
