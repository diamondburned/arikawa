// Package json allows for different implementations of JSON serializing, as
// well as extra optional types needed.
package json

import (
	"encoding/json"
	"io"
)

type Driver interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error

	DecodeStream(r io.Reader, v interface{}) error
	EncodeStream(w io.Writer, v interface{}) error
}

type DefaultDriver struct{}

func (d DefaultDriver) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (d DefaultDriver) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (d DefaultDriver) DecodeStream(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func (d DefaultDriver) EncodeStream(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// Default is the default JSON driver, which uses encoding/json.
var Default Driver = DefaultDriver{}

// Marshal uses the default driver.
func Marshal(v interface{}) ([]byte, error) {
	return Default.Marshal(v)
}

// Unmarshal uses the default driver.
func Unmarshal(data []byte, v interface{}) error {
	return Default.Unmarshal(data, v)
}

// DecodeStream uses the default driver.
func DecodeStream(r io.Reader, v interface{}) error {
	return Default.DecodeStream(r, v)
}

// EncodeStream uses the default driver.
func EncodeStream(w io.Writer, v interface{}) error {
	return Default.EncodeStream(w, v)
}
