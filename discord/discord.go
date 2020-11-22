// Package discord provides common structures that the whole repository uses. It
// does not (and should not) contain API-specific structures, or WS-specific
// structures.
package discord

// HasFlag is returns true if has is in the flag. In other words, it checks if
// has is OR'ed into flag. This function could be used for different constants
// such as Permission.
func HasFlag(flag, has uint64) bool {
	return flag&has == has
}
