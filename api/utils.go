package api

// min returns the smaller of the two passed numbers.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
