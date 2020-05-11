package option

import "strconv"

// EnumNull is the value used to represent JSON null.
// It should never be used as a value, as it won't get serialized as such.
const EnumNull = -1

// Enum is a nullable version of a uint8.
// Enum values should only consist of positive values, as negative values are reserved for internal constants, such as
// EnumNull.
// This also mean that only 7 of the 8 Bit will be available for storage.
type Enum int8

// Int8ToJSON converts the passed Enum to a byte slice with it's JSON representation.
func EnumToJSON(i Enum) []byte {
	if i == EnumNull {
		return []byte("null")
	} else {
		return []byte(strconv.Itoa(int(i)))
	}
}

// Int8FromJSON decodes the Enum stored as JSON src the passed byte slice.
func EnumFromJSON(b []byte) (Enum, error) {
	s := string(b)

	if s == "null" {
		return EnumNull, nil
	} else {
		i, err := strconv.ParseUint(s, 10, 7)
		return Enum(i), err
	}
}
