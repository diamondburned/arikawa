// Package discord provides common structures that the whole repository uses. It
// does not (and should not) contain API-specific structures, or WS-specific
// structures.
package discord

import "fmt"

// HasFlag is returns true if has is in the flag. In other words, it checks if
// has is OR'ed into flag. This function could be used for different constants
// such as Permission.
func HasFlag(flag, has uint64) bool {
	return flag&has == has
}

// OverboundError is an error that's returned if any value is too long.
type OverboundError struct {
	Count int
	Max   int

	Thing string
}

var _ error = (*OverboundError)(nil)

func (e *OverboundError) Error() string {
	if e.Thing == "" {
		return fmt.Sprintf("Overbound error: %d > %d", e.Count, e.Max)
	}

	return fmt.Sprintf(e.Thing+" overbound: %d > %d", e.Count, e.Max)
}
