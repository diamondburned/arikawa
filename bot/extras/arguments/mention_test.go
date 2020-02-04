package arguments

import (
	"testing"

	"github.com/diamondburned/arikawa/discord"
)

func TestMention(t *testing.T) {
	var (
		c ChannelMention
		u UserMention
		r RoleMention
	)

	type mention interface {
		Parse(arg string) error
		ID() discord.Snowflake
		Mention() string
	}

	var tests = []struct {
		mention
		str string
		id  discord.Snowflake
	}{
		{&c, "<#123123>", 123123},
		{&r, "<@&23321>", 23321},
		{&u, "<@123123>", 123123},
	}

	for _, test := range tests {
		if err := test.Parse(test.str); err != nil {
			t.Fatal("Expected", test.id, "error:", err)
		}

		if id := test.ID(); id != test.id {
			t.Fatal("Expected", test.id, "got", id)
		}

		if mention := test.Mention(); mention != test.str {
			t.Fatal("Expected", test.str, "got", mention)
		}
	}
}
