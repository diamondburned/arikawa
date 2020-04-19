// Package httpdriver provides interfaces and implementations of a simple HTTP
// client.
package httpdriver

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// NoContent is the status code for HTTP 204, or http.StatusNoContent.
const NoContent = 204

// Client is a generic interface used as an adapter allowing for custom HTTP
// client implementations, such as fasthttp.
type Client interface {
	NewRequest(ctx context.Context, method, url string) (Request, error)
	Do(req Request) (Response, error)
}

// Request is a generic interface for a normal HTTP request. It should be
// constructed using (Requester).NewRequest().
type Request interface {
	// GetPath should return the URL path, for example "/endpoint/thing".
	GetPath() string
	// GetContext should return the same context that's passed into NewRequest.
	// For implementations that don't support this, it can remove a
	// context.Background().
	GetContext() context.Context
	// AddHeader appends headers.
	AddHeader(http.Header)
	// AddQuery appends URL query values.
	AddQuery(url.Values)
	// WithBody should automatically close the ReadCloser on finish. This is
	// similar to the stdlib's Request behavior.
	WithBody(io.ReadCloser)
}

// Response is returned from (Requester).DoContext().
type Response interface {
	GetStatus() int
	GetHeader() http.Header
	// Body's ReadCloser will always be closed when done, unless DoContext()
	// returns an error.
	GetBody() io.ReadCloser
}

// OptHeader returns the response header, or nil if from is nil.
func OptHeader(from Response) http.Header {
	if from == nil {
		return nil
	}
	return from.GetHeader()
}
