package option

// ================================ Uint ================================

// Uint is the option type for unsigned integers (uint).
type Uint *uint

// ZeroUint is a Uint with 0 as value.
var ZeroUint = NewUint(0)

// NewUint creates a new Uint using the value of the passed uint.
func NewUint(u uint) Uint { return &u }

// ================================ Int ================================

// Int is the option type for integers (int).
type Int *int

// ZeroInt is an Int with 0 as value.
var ZeroInt = NewInt(0)

// NewInt creates a new Int using the value of the passed int.
func NewInt(i int) Int { return &i }
