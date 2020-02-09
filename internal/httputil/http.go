package httputil

import (
	"net/http"
)

type TransportWrapper struct {
	Default http.RoundTripper
	Pre     func(*http.Request) error
	Cancel  func(*http.Request, error)
	Post    func(*http.Response) error
}

var _ http.RoundTripper = (*TransportWrapper)(nil)

func NewTransportWrapper() *TransportWrapper {
	return &TransportWrapper{
		Default: http.DefaultTransport,
		Pre:     func(*http.Request) error { return nil },
		Cancel:  func(*http.Request, error) {},
		Post:    func(*http.Response) error { return nil },
	}
}

func (c *TransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := c.Pre(req); err != nil {
		return nil, err
	}

	r, err := c.Default.RoundTrip(req)
	if err != nil {
		c.Cancel(req, err)
		return nil, err
	}

	if err := c.Post(r); err != nil {
		return nil, err
	}

	return r, nil
}
