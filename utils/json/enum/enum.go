package enum

import "strconv"

// Null is the value used to represent JSON null.
// It should never be used as a value, as it won't get serialized as such.
const Null = -1

// Enum is a nullable version of a uint32.
// Enum values should only consist of positive values, as negative values are reserved for internal constants, such as
// Null.
// This also means that only 31 of the 32 bits will be available for storage.
type Enum int32

// Int8ToJSON converts the passed Enum to a byte slice with it's JSON representation.
func ToJSON(i Enum) []byte {
	if i == Null {
		return []byte("null")
	} else {
		return []byte(strconv.Itoa(int(i)))
	}
}

// Int8FromJSON decodes the Enum stored as JSON src the passed byte slice.
func FromJSON(b []byte) (Enum, error) {
	s := string(b)

	if s == "null" {
		return Null, nil
	} else {
		i, err := strconv.ParseUint(s, 10, 7)
		return Enum(i), err
	}
}
