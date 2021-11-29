package httputil

import (
	"fmt"
	"strconv"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

// JSONError is returned if the request responds with an invalid JSON.
type JSONError struct {
	err error
}

func (j JSONError) Error() string {
	return "JSON decoding failed: " + j.err.Error()
}

func (j JSONError) Unwrap() error {
	return j.err
}

// RequestError is returned if the request fails to be done, i.e. the server is
// never reached.
type RequestError struct {
	err error
}

func (r RequestError) Error() string {
	return "request failed: " + r.err.Error()
}

func (r RequestError) Unwrap() error {
	return r.err
}

// HTTPError is returned if the server responds successfully with an error of
// any kind.
type HTTPError struct {
	Status int    `json:"-"`
	Body   []byte `json:"-"`

	Code    ErrorCode `json:"code"`
	Errors  json.Raw  `json:"errors,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (err HTTPError) Error() string {
	switch {
	case err.Errors != nil:
		return fmt.Sprintf("Discord %d error: %s: %s", err.Status, err.Message, err.Errors)

	case err.Message != "":
		return fmt.Sprintf("Discord %d error: %s", err.Status, err.Message)

	case err.Code > 0:
		return fmt.Sprintf("Discord returned status %d error code %d",
			err.Status, err.Code)

	case len(err.Body) > 0:
		return fmt.Sprintf("Discord returned status %d body %s",
			err.Status, string(err.Body))

	default:
		return "Discord returned status " + strconv.Itoa(err.Status)
	}
}

type ErrorCode uint
