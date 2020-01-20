// +build unit

package bot

import "testing"

func TestNameFlag(t *testing.T) {
	type entry struct {
		Name   string
		Expect NameFlag
		String string
	}

	var entries = []entry{{
		Name:   "AーEcho",
		Expect: AdminOnly,
	}, {
		Name:   "RAーGC",
		Expect: Raw | AdminOnly,
	}}

	for _, entry := range entries {
		var f, _ = ParseFlag(entry.Name)
		if !f.Is(entry.Expect) {
			t.Fatalf("unexpected expectation for %s: %v", entry.Name, f)
		}
	}
}
