package option

import "strconv"

// ================================ Bool ================================

// Bool is the option type for bool.
type Bool *bool

var (
	True  = newBool(true)
	False = newBool(false)
)

// newBool creates a new Bool with the value of the passed bool.
func newBool(b bool) Bool { return &b }

// ================================ NullableBool ================================

// NullableBool is the nullable type for bool.
type NullableBool = *nullableBool

type nullableBool struct {
	Val  bool
	Init bool
}

var (
	// NullBool serializes to JSON null.
	NullBool     = &nullableBool{}
	NullableTrue = &nullableBool{
		Val:  true,
		Init: true,
	}
	NullableFalse = &nullableBool{
		Val:  false,
		Init: true,
	}
)

func (b nullableBool) MarshalJSON() ([]byte, error) {
	if !b.Init {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatBool(b.Val)), nil
}

func (b *nullableBool) UnmarshalJSON(json []byte) (err error) {
	s := string(json)

	if s == "null" {
		b.Init = false
		return
	}

	b.Val, err = strconv.ParseBool(s)

	return
}
