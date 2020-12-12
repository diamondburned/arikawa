package httputil

import (
	"fmt"
	"strconv"
)

type JSONError struct {
	err error
}

func (j JSONError) Error() string {
	return "JSON decoding failed: " + j.err.Error()
}

func (j JSONError) Unwrap() error {
	return j.err
}

type RequestError struct {
	err error
}

func (r RequestError) Error() string {
	return "request failed: " + r.err.Error()
}

func (r RequestError) Unwrap() error {
	return r.err
}

type HTTPError struct {
	Status int    `json:"-"`
	Body   []byte `json:"-"`

	Code    ErrorCode `json:"code"`
	Message string    `json:"message,omitempty"`
}

func (err HTTPError) Error() string {
	switch {
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
