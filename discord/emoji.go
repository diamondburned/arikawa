package discord

import (
	"net/url"
	"strings"
	"time"
)

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
	// This may be false due to loss of Server Boosts.
	//
	// This field is only available for custom emojis.
	Available bool `json:"available,omitempty"`
}

// IsCustom returns whether the emoji is a custom emoji.
func (e Emoji) IsCustom() bool {
	return e.ID.IsValid()
}

// IsUnicode returns whether the emoji is a unicode emoji.
func (e Emoji) IsUnicode() bool {
	return !e.IsCustom()
}

// CreatedAt returns a time object representing when the emoji was created.
//
// This will only work for custom emojis.
func (e Emoji) CreatedAt() time.Time {
	return e.ID.Time()
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
	if e.IsUnicode() {
		return ""
	}

	if t == AutoImage {
		return e.EmojiURL()
	}

	return "https://cdn.discordapp.com/emojis/" + t.format(e.ID.String())
}

// APIEmoji represents an emoji identifier string formatted to be used with the
// API. It is formatted using Emoji's APIString method as well as the
// NewCustomEmoji function. If the emoji is a stock Unicode emoji, then this
// string contains it. Otherwise, it is formatted like "emoji_name:123123123",
// where "123123123" is the emoji ID.
type APIEmoji string

// NewCustomEmoji creates a new Emoji using a custom guild emoji as base.
// Unicode emojis should be directly converted.
func NewCustomEmoji(id EmojiID, name string) APIEmoji {
	return APIEmoji(name + ":" + id.String())
}

// PathString returns the APIEmoji as a path-encoded string.
func (e APIEmoji) PathString() string {
	return url.PathEscape(string(e))
}

// APIString returns a string usable for sending over to the API.
func (e Emoji) APIString() APIEmoji {
	if e.IsUnicode() {
		return APIEmoji(e.Name)
	}

	return NewCustomEmoji(e.ID, e.Name)
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
