package json

type (
	OptionBool   = *bool
	OptionString = *string
	OptionUint   = *uint
	OptionInt    = *int
)

var (
	True  = getBool(true)
	False = getBool(false)

	ZeroUint = Uint(0)
	ZeroInt  = Int(0)

	EmptyString = String("")
)

func Uint(u uint) OptionUint {
	return &u
}

func Int(i int) OptionInt {
	return &i
}

func String(s string) OptionString {
	return &s
}

func getBool(Bool bool) OptionBool {
	return &Bool
}
