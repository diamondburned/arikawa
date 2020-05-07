package json

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

func (m Raw) String() string {
	return string(m)
}
