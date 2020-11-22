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
type NullableBool = *NullableBoolData

type NullableBoolData struct {
	Val  bool
	Init bool
}

var (
	// NullBool serializes to JSON null.
	NullBool     = &NullableBoolData{}
	NullableTrue = &NullableBoolData{
		Val:  true,
		Init: true,
	}
	NullableFalse = &NullableBoolData{
		Val:  false,
		Init: true,
	}
)

func (b NullableBoolData) MarshalJSON() ([]byte, error) {
	if !b.Init {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatBool(b.Val)), nil
}

func (b *NullableBoolData) UnmarshalJSON(json []byte) (err error) {
	s := string(json)

	if s == "null" {
		*b = *NullBool
		return
	}

	b.Val, err = strconv.ParseBool(s)
	b.Init = true

	return
}
