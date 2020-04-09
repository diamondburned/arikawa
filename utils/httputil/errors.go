package httputil

import (
	"fmt"
	"strconv"
)

type JSONError struct {
	error
}

type RequestError struct {
	error
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
		return "Discord error: " + err.Message

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
