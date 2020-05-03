package httpdriver

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultClient implements Client and wraps around the stdlib Client.
type DefaultClient http.Client

var _ Client = (*DefaultClient)(nil)

// WrapClient wraps around the standard library's http.Client and returns an
// implementation that's compatible with the Client driver interface.
func WrapClient(client http.Client) Client {
	return DefaultClient(client)
}

// NewClient creates a new client around the standard library's http.Client. The
// client will have a timeout of 10 seconds.
func NewClient() Client {
	return WrapClient(http.Client{
		Timeout: 10 * time.Second,
	})
}

func (d DefaultClient) NewRequest(ctx context.Context, method, url string) (Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	return (*DefaultRequest)(r), nil
}

func (d DefaultClient) Do(req Request) (Response, error) {
	// Implementations can safely assert this.
	request := req.(*DefaultRequest)

	r, err := (*http.Client)(&d).Do((*http.Request)(request))
	if err != nil {
		return nil, err
	}

	return (*DefaultResponse)(r), nil
}

// DefaultRequest wraps around the stdlib Request and satisfies the Request
// interface.
type DefaultRequest http.Request

var _ Request = (*DefaultRequest)(nil)

func (r *DefaultRequest) GetPath() string {
	return r.URL.Path
}

func (r *DefaultRequest) GetContext() context.Context {
	return (*http.Request)(r).Context()
}

func (r *DefaultRequest) AddQuery(values url.Values) {
	var qs = r.URL.Query()
	for k, v := range values {
		qs[k] = append(qs[k], v...)
	}

	r.URL.RawQuery = qs.Encode()
}

func (r *DefaultRequest) AddHeader(header http.Header) {
	for key, values := range header {
		r.Header[key] = append(r.Header[key], values...)
	}
}

func (r *DefaultRequest) WithBody(body io.ReadCloser) {
	r.Body = body
}

// DefaultResponse wraps around the stdlib Response and satisfies the Response
// interface.
type DefaultResponse http.Response

var _ Response = (*DefaultResponse)(nil)

func (r *DefaultResponse) GetStatus() int {
	return r.StatusCode
}

func (r *DefaultResponse) GetHeader() http.Header {
	return r.Header
}

func (r *DefaultResponse) GetBody() io.ReadCloser {
	return r.Body
}
