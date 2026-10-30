package api

import (
	"encoding/json"
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
)

func TestRoleMemberCountsUnmarshalJSON(t *testing.T) {
	const sample = `{
		"613425648685547541": 1337,
		"1409696176629878905": 2,
		"697138785317814292": 67
	}`

	var counts RoleMemberCounts
	if err := json.Unmarshal([]byte(sample), &counts); err != nil {
		t.Fatal("unexpected error:", err)
	}

	want := RoleMemberCounts{
		discord.RoleID(613425648685547541): 1337,
		discord.RoleID(1409696176629878905): 2,
		discord.RoleID(697138785317814292):  67,
	}

	if len(counts) != len(want) {
		t.Fatalf("unexpected length %d, want %d", len(counts), len(want))
	}

	for id, n := range want {
		if counts[id] != n {
			t.Fatalf("count for role %s: got %d, want %d", id, counts[id], n)
		}
	}
}
