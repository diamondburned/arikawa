package json

import (
	"bytes"
	"strconv"
)

type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

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

func (m *Raw) UnmarshalJSON(data []byte) error {
	*m = append((*m)[0:0], data...)
	return nil
}

func (m Raw) UnmarshalTo(v interface{}) error {
	// Leave as nil.
	if len(m) == 0 {
		return nil
	}
	return Unmarshal(m, v)
}

func (m Raw) String() string {
	return string(m)
}

// AlwaysString would always unmarshal into a string, from any JSON type. Quotes
// will be stripped.
type AlwaysString string

func (m *AlwaysString) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, `"`)
	*m = AlwaysString(data)
	return nil
}

func (m AlwaysString) Int() (int, error) {
	return strconv.Atoi(string(m))
}
