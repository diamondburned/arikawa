package api

import "testing"

func Test_min(t *testing.T) {
	testCases := []struct {
		name   string
		a, b   int
		expect int
	}{
		{
			name:   "first smaller",
			a:      1,
			b:      2,
			expect: 1,
		},
		{
			name:   "both equal",
			a:      1,
			b:      1,
			expect: 1,
		},
		{
			name:   "last smaller",
			a:      2,
			b:      1,
			expect: 1,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			actual := min(c.a, c.b)
			if c.expect != actual {
				t.Errorf("expected min(%d, %d) to return %d, but did %d", c.a, c.b, c.expect, actual)
			}
		})
	}
}
