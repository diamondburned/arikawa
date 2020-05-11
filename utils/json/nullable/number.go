package nullable

type (
	// Uint is a nullable version of an unsigned integer (uint).
	Uint *uint
	// Int is a nullable version of an integer (int).
	Int *int
)

var (
	// ZeroUint is a Uint with 0 as value.
	ZeroUint = NewUint(0)
	// ZeroInt is an Int with 0 as value.
	ZeroInt = NewInt(0)
)

// NewUint creates a new Uint using the value of the passed uint.
func NewUint(u uint) Uint {
	return &u
}

// NewInt creates a new Int using the value of the passed int.
func NewInt(i int) Int {
	return &i
}
