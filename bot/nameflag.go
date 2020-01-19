package bot

import "strings"

type NameFlag uint64

const FlagSeparator = 'ー'

const (
	None NameFlag = 1 << iota

	// These flags only apply to messageCreate events.

	Raw       // R
	AdminOnly // A
)

func ParseFlag(name string) (NameFlag, string) {
	parts := strings.SplitN(name, string(FlagSeparator), 2)
	if len(parts) != 2 {
		return 0, name
	}

	var f NameFlag

	for _, r := range parts[0] {
		switch r {
		case 'R':
			f |= Raw
		case 'A':
			f |= AdminOnly
		}
	}

	return f, parts[1]
}

func (f NameFlag) Is(flag NameFlag) bool {
	return f&flag != 0
}
