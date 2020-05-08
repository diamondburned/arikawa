package httputil

import (
	"io"
	"net/http"

	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/utils/json"
)

type RequestOption func(httpdriver.Request) error
type ResponseFunc func(httpdriver.Request, httpdriver.Response) error

func PrependOptions(opts []RequestOption, prepend ...RequestOption) []RequestOption {
	if len(opts) == 0 {
		return prepend
	}
	return append(prepend, opts...)
}

func JSONRequest(r httpdriver.Request) error {
	r.AddHeader(http.Header{
		"Content-Type": {"application/json"},
	})
	return nil
}

func MultipartRequest(r httpdriver.Request) error {
	r.AddHeader(http.Header{
		"Content-Type": {"multipart/form-data"},
	})
	return nil
}

func WithHeaders(headers http.Header) RequestOption {
	return func(r httpdriver.Request) error {
		r.AddHeader(headers)
		return nil
	}
}

func WithContentType(ctype string) RequestOption {
	return func(r httpdriver.Request) error {
		r.AddHeader(http.Header{
			"Content-Type": {ctype},
		})
		return nil
	}
}

func WithSchema(schema SchemaEncoder, v interface{}) RequestOption {
	return func(r httpdriver.Request) error {
		params, err := schema.Encode(v)
		if err != nil {
			return err
		}

		r.AddQuery(params)
		return nil
	}
}

func WithBody(body io.ReadCloser) RequestOption {
	return func(r httpdriver.Request) error {
		r.WithBody(body)
		return nil
	}
}

// WithJSONBody inserts a JSON body into the request. This ignores JSON errors.
func WithJSONBody(v interface{}) RequestOption {
	if v == nil {
		return func(httpdriver.Request) error {
			return nil
		}
	}

	var rp, wp = io.Pipe()
	var err error

	go func() {
		err = json.EncodeStream(wp, v)
		wp.Close()
	}()

	return func(r httpdriver.Request) error {
		// TODO: maybe do something to this?
		if err != nil {
			return err
		}

		r.AddHeader(http.Header{
			"Content-Type": {"application/json"},
		})
		r.WithBody(rp)
		return nil
	}
}
