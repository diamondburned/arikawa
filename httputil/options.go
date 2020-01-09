package httputil

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/diamondburned/arikawa/json"
)

type RequestOption func(*http.Request) error

func JSONRequest(r *http.Request) error {
	r.Header.Set("Content-Type", "application/json")
	return nil
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
		r.Body = body
		return nil
	}
}

func WithJSONBody(json json.Driver, v interface{}) RequestOption {
	if v == nil {
		return func(*http.Request) error {
			return nil
		}
	}

	var buf bytes.Buffer
	var err error

	go func() {
		err = json.EncodeStream(&buf, v)
	}()

	return func(r *http.Request) error {
		if err != nil {
			return err
		}

		r.Header.Set("Content-Type", "application/json")
		r.Body = ioutil.NopCloser(&buf)
		return nil
	}
}
