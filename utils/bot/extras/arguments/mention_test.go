package arguments

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
)

func TestChannelMention(t *testing.T) {
	test := new(ChannelMention)
	str := "<#123123>"
	var id discord.ChannelID = 123123

	if err := test.Parse(str); err != nil {
		t.Fatal("Expected", id, "error:", err)
	}

	if actualID := test.ID(); actualID != id {
		t.Fatal("Expected", id, "got", id)
	}

	if mention := test.Mention(); mention != str {
		t.Fatal("Expected", str, "got", mention)
	}
}

func TestUserMention(t *testing.T) {
	test := new(UserMention)
	str := "<@123123>"
	var id discord.UserID = 123123

	if err := test.Parse(str); err != nil {
		t.Fatal("Expected", id, "error:", err)
	}

	if actualID := test.ID(); actualID != id {
		t.Fatal("Expected", id, "got", id)
	}

	if mention := test.Mention(); mention != str {
		t.Fatal("Expected", str, "got", mention)
	}
}

func TestRoleMention(t *testing.T) {
	test := new(RoleMention)
	str := "<@&123123>"
	var id discord.RoleID = 123123

	if err := test.Parse(str); err != nil {
		t.Fatal("Expected", id, "error:", err)
	}

	if actualID := test.ID(); actualID != id {
		t.Fatal("Expected", id, "got", id)
	}

	if mention := test.Mention(); mention != str {
		t.Fatal("Expected", str, "got", mention)
	}
}
