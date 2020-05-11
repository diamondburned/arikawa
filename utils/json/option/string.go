package option

// String is the option type for strings.
type String *string

// EmptyString is a zero-length string.
var EmptyString = NewString("")

// NewString creates a new String with the value of the passed string.
func NewString(s string) String { return &s }
