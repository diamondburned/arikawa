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
type NullableString = *nullableString

type nullableString struct {
	Val  string
	Init bool
}

// NullBool serializes to JSON null.
var NullString = &nullableString{}

// NewNullableString creates a new non-null NullableString with the value of the passed string.
func NewNullableString(v string) NullableString {
	return &nullableString{
		Val:  v,
		Init: true,
	}
}

func (s nullableString) MarshalJSON() ([]byte, error) {
	if !s.Init {
		return []byte("null"), nil
	}

	return []byte("\"" + s.Val + "\""), nil
}

func (s *nullableString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		s.Init = false
		return nil
	}

	return json.Unmarshal(b, &s.Val)
}
