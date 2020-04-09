package httputil

import (
	"net/http"
)

type TransportWrapper struct {
	Default http.RoundTripper
	Pre     func(*http.Request) error
	Post    func(*http.Request, *http.Response) error
}

var _ http.RoundTripper = (*TransportWrapper)(nil)

func NewTransportWrapper() *TransportWrapper {
	return &TransportWrapper{
		Default: http.DefaultTransport,
		Pre:     func(*http.Request) error { return nil },
		Post:    func(*http.Request, *http.Response) error { return nil },
	}
}

func (c *TransportWrapper) RoundTrip(req *http.Request) (r *http.Response, err error) {
	if err := c.Pre(req); err != nil {
		return nil, err
	}

	r, err = c.Default.RoundTrip(req)

	// Call Post regardless of error:
	if postErr := c.Post(req, r); postErr != nil {
		return r, postErr
	}

	return r, err
}
