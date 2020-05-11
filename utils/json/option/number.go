package option

type (
	// Uint is the option type for unsigned integers (uint).
	Uint *uint
	// Int is the option type for integers (int).
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
