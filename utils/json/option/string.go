package option

import (
	"encoding/json"
)

// ================================ String ================================

// String is the option type for strings.
type String *string

// NewString creates a new String with the value of the passed string.
func NewString(s string) String { return &s }

// ================================ NullableString ================================

// NullableString is a nullable version of a string.
type NullableString = *NullableStringData

type NullableStringData struct {
	Val  string
	Init bool
}

// NullString serializes to JSON null.
var NullString = &NullableStringData{}

// NewNullableString creates a new non-null NullableString with the value of the passed string.
func NewNullableString(v string) NullableString {
	return &NullableStringData{
		Val:  v,
		Init: true,
	}
}

func (s NullableStringData) MarshalJSON() ([]byte, error) {
	if !s.Init {
		return []byte("null"), nil
	}
	return json.Marshal(s.Val)
}

func (s *NullableStringData) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = *NullString
		return nil
	}

	s.Init = true
	return json.Unmarshal(b, &s.Val)
}
