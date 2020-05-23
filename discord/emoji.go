package discord

import "strings"

type Emoji struct {
	ID   Snowflake `json:"id,string"` // NullSnowflake for unicode emojis
	Name string    `json:"name"`

	// These fields are optional

	RoleIDs []Snowflake `json:"roles,omitempty"`
	User    User        `json:"user,omitempty"`

	RequireColons bool `json:"require_colons,omitempty"`
	Managed       bool `json:"managed,omitempty"`
	Animated      bool `json:"animated,omitempty"`
}

// EmojiURL returns the URL of the emoji and auto-detects a suitable type.
//
// This will only work for custom emojis.
func (e Emoji) EmojiURL() string {
	if e.Animated {
		return e.EmojiURLWithType(GIFImage)
	}

	return e.EmojiURLWithType(PNGImage)
}

// EmojiURLWithType returns the URL to the emoji's image.
//
// This will only work for custom emojis.
//
// Supported ImageTypes: PNG, GIF
func (e Emoji) EmojiURLWithType(t ImageType) string {
	if e.ID == NullSnowflake {
		return ""
	}

	if t == AutoImage {
		return e.EmojiURL()
	}

	return "https://cdn.discordapp.com/emojis/" + t.format(e.ID.String())
}

// APIString returns a string usable for sending over to the API.
func (e Emoji) APIString() string {
	if !e.ID.Valid() {
		return e.Name // is unicode
	}

	return e.Name + ":" + e.ID.String()
}

// String formats the string like how the client does.
func (e Emoji) String() string {
	if e.ID == 0 {
		return e.Name
	}

	var parts = [3]string{
		"", e.Name, e.ID.String(),
	}

	if e.Animated {
		parts[0] = "a"
	}

	return "<" + strings.Join(parts[:], ":") + ">"
}
