package discord

import "strings"

// https://discord.com/developers/docs/resources/emoji#emoji-object
type Emoji struct {
	// ID is the ID of the Emoji.
	// The ID will be NullSnowflake, if the Emoji is a Unicode emoji.
	ID EmojiID `json:"id"`
	// Name is the name of the emoji.
	Name string `json:"name"`

	// RoleIDs are the roles the emoji is whitelisted to.
	//
	// This field is only available for custom emojis.
	RoleIDs []RoleID `json:"roles,omitempty"`
	// User is the user that created the emoji.
	//
	// This field is only available for custom emojis.
	User User `json:"user,omitempty"`

	// RequireColons specifies whether the emoji must be wrapped in colons.
	//
	// This field is only available for custom emojis.
	RequireColons bool `json:"require_colons,omitempty"`
	// Managed specifies whether the emoji is managed.
	//
	// This field is only available for custom emojis.
	Managed bool `json:"managed,omitempty"`
	// Animated specifies whether the emoji is animated.
	//
	// This field is only available for custom emojis.
	Animated bool `json:"animated,omitempty"`
	// Available specifies whether the emoji can be used.
	// This may be false tue to loss of Server Boosts.
	//
	// This field is only available for custom emojis.
	Available bool `json:"available,omitempty"`
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
	if e.ID.IsNull() {
		return ""
	}

	if t == AutoImage {
		return e.EmojiURL()
	}

	return "https://cdn.discordapp.com/emojis/" + t.format(e.ID.String())
}

// APIString returns a string usable for sending over to the API.
func (e Emoji) APIString() string {
	if !e.ID.IsValid() {
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
