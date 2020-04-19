package httputil

import (
	"io"
	"net/http"

	"github.com/diamondburned/arikawa/utils/json"
)

type RequestOption func(*http.Request) error

func JSONRequest(r *http.Request) error {
	r.Header.Set("Content-Type", "application/json")
	return nil
}

func MultipartRequest(r *http.Request) error {
	r.Header.Set("Content-Type", "multipart/form-data")
	return nil
}

func WithHeaders(headers http.Header) RequestOption {
	return func(r *http.Request) error {
		for key, values := range headers {
			r.Header[key] = append(r.Header[key], values...)
		}
		return nil
	}
}

func WithContentType(ctype string) RequestOption {
	return func(r *http.Request) error {
		r.Header.Set("Content-Type", ctype)
		return nil
	}
}

func WithSchema(schema SchemaEncoder, v interface{}) RequestOption {
	return func(r *http.Request) error {
		params, err := schema.Encode(v)
		if err != nil {
			return err
		}

		var qs = r.URL.Query()
		for k, v := range params {
			qs[k] = append(qs[k], v...)
		}

		r.URL.RawQuery = qs.Encode()
		return nil
	}
}

func WithBody(body io.ReadCloser) RequestOption {
	return func(r *http.Request) error {
		// tee := io.TeeReader(body, os.Stderr)
		// r.Body = ioutil.NopCloser(tee)
		r.Body = body
		r.ContentLength = -1
		return nil
	}
}

func WithJSONBody(json json.Driver, v interface{}) RequestOption {
	if v == nil {
		return func(*http.Request) error {
			return nil
		}
	}

	var err error
	var rp, wp = io.Pipe()

	go func() {
		err = json.EncodeStream(wp, v)
		wp.Close()
	}()

	return func(r *http.Request) error {
		if err != nil {
			return err
		}

		r.Header.Set("Content-Type", "application/json")
		r.Body = rp
		return nil
	}
}
