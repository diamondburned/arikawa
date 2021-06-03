package httputil

import (
	"io"

	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
)

// This file contains mistakes.

// httpResponse wraps around a httpdriver.Response to provide a custom body.
type httpResponse struct {
	httpdriver.Response
	body io.ReadCloser
}

func wrapCancelableResponse(r httpdriver.Response, cancel func()) httpdriver.Response {
	body := bodyCloser{
		ReadCloser: r.GetBody(),
		close:      cancel,
	}
	return httpResponse{
		Response: r,
		body:     body,
	}
}

func (resp httpResponse) GetBody() io.ReadCloser {
	return resp.body
}

// bodyCloser wraps around a body to add an additional close callback.
type bodyCloser struct {
	io.ReadCloser
	close func()
}

func (body bodyCloser) Close() error {
	err := body.ReadCloser.Close()
	body.close()
	return err
}
