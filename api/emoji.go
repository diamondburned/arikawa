package api

import (
	"strings"

	"git.sr.ht/~diamondburned/arikawa/discord"
)

type Emoji struct {
	ID   discord.Snowflake `json:"id"` // 0 for Unicode emojis
	Name string            `json:"name"`

	// These fields are optional

	RoleIDs []discord.Snowflake `json:"roles,omitempty"`
	User    User                `json:"user,omitempty"`

	RequireColons bool `json:"require_colons,omitempty"`
	Managed       bool `json:"managed,omitempty"`
	Animated      bool `json:"animated,omitempty"`
}

// EmojiAPI is a special format that the API wants.
type EmojiAPI = string

func FormatEmojiAPI(id discord.Snowflake, name string) string {
	if id == 0 {
		return name
	}

	return id.String() + ":" + name
}

// APIString returns a string usable for sending over to the API.
func (e Emoji) APIString() EmojiAPI {
	if e.ID == 0 {
		return e.Name // is unicode
	}

	return e.ID.String() + ":" + e.Name
}

// String formats the string like how the client does.
func (e Emoji) String() string {
	if e.ID == 0 {
		return e.Name
	}

	var parts = []string{
		"", e.Name, e.ID.String(),
	}

	if e.Animated {
		parts[0] = "a"
	}

	return "<" + strings.Join(parts, ":") + ">"
}
