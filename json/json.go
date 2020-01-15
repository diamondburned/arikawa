// Package json allows for different implementations of JSON serializing, as
// well as extra optional types needed.
package json

import (
	"encoding/json"
	"io"
)

type (
	OptionBool   = *bool
	OptionString = *string
	OptionUint   = *uint
	OptionInt    = *int
)

var (
	True  = getBool(true)
	False = getBool(false)

	ZeroUint = Uint(0)
	ZeroInt  = Int(0)

	EmptyString = String("")
)

func Uint(u uint) OptionUint {
	return &u
}

func Int(i int) OptionInt {
	return &i
}

func String(s string) OptionString {
	return &s
}

func getBool(Bool bool) OptionBool {
	return &Bool
}

//

// Raw is a raw encoded JSON value. It implements Marshaler and Unmarshaler and
// can be used to delay JSON decoding or precompute a JSON encoding. It's taken
// from encoding/json.
type Raw []byte

// Raw returns m as the JSON encoding of m.
func (m Raw) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

func (m Raw) String() string {
	return string(m)
}

//

type Driver interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error

	DecodeStream(r io.Reader, v interface{}) error
	EncodeStream(w io.Writer, v interface{}) error
}

type Default struct{}

func (d Default) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (d Default) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (d Default) DecodeStream(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func (d Default) EncodeStream(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}
