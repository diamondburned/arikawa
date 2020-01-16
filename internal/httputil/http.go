package httputil

import (
	"net/http"
)

type TransportWrapper struct {
	http.RoundTripper

	Pre  func(*http.Request) error
	Post func(*http.Response) error
}

func NewTransportWrapper() *TransportWrapper {
	return &TransportWrapper{
		RoundTripper: http.DefaultTransport,

		Pre:  func(*http.Request) error { return nil },
		Post: func(*http.Response) error { return nil },
	}
}

func (c *TransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := c.Pre(req); err != nil {
		return nil, err
	}

	r, err := c.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if err := c.Post(r); err != nil {
		return nil, err
	}

	return r, nil
}
