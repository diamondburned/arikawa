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
