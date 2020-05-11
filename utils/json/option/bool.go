package option

// Bool is the option type for bool.
type Bool *bool

var (
	True  = newBool(true)
	False = newBool(false)
)

// newBool creates a new Bool with the value of the passed bool.
func newBool(b bool) Bool { return &b }
