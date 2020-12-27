package intmath

// Min returns the smaller of the two passed numbers.
func Min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
