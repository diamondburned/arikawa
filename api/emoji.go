package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

// Emoji is the API format of a regular Emoji, both Unicode or custom.
type Emoji = string

// NewCustomEmoji creates a new Emoji using a custom guild emoji as
// base.
// Unicode emojis should be directly passed to the function using Emoji.
func NewCustomEmoji(id discord.Snowflake, name string) Emoji {
	return name + ":" + id.String()
}

// Emojis returns a list of emoji objects for the given guild.
func (c *Client) Emojis(guildID discord.Snowflake) ([]discord.Emoji, error) {
	var emjs []discord.Emoji
	return emjs, c.RequestJSON(&emjs, "GET", EndpointGuilds+guildID.String()+"/emojis")
}

// Emoji returns an emoji object for the given guild and emoji IDs.
func (c *Client) Emoji(guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {
	var emj *discord.Emoji
	return emj, c.RequestJSON(&emj, "GET",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String())
}

// https://discord.com/developers/docs/resources/emoji#create-guild-emoji-json-params
type CreateEmojiData struct {
	// Name is the name of the emoji.
	Name string `json:"name"`
	// Image is the the 128x128 emoji image.
	Image Image `json:"image"`
	// Roles are the roles for which this emoji will be whitelisted.
	Roles *[]discord.Snowflake `json:"roles,omitempty"`
}

// CreateEmoji creates a new emoji in the guild. This endpoint requires
// MANAGE_EMOJIS. ContentType must be "image/jpeg", "image/png", or
// "image/gif". However, ContentType can also be automatically detected
// (though shouldn't be relied on).
// Emojis and animated emojis have a maximum file size of 256kb.
func (c *Client) CreateEmoji(
	guildID discord.Snowflake, data CreateEmojiData) (*discord.Emoji, error) {

	// Max 256KB
	if err := data.Image.Validate(256 * 1000); err != nil {
		return nil, err
	}

	var emj *discord.Emoji
	return emj, c.RequestJSON(
		&emj, "POST",
		EndpointGuilds+guildID.String()+"/emojis",
		httputil.WithJSONBody(data),
	)
}

// https://discord.com/developers/docs/resources/emoji#modify-guild-emoji-json-params
type ModifyEmojiData struct {
	// Name is the name of the emoji.
	Name string `json:"name,omitempty"`
	// Roles are the roles to which this emoji will be whitelisted.
	Roles *[]discord.Snowflake `json:"roles,omitempty"`
}

// ModifyEmoji changes an existing emoji. This requires MANAGE_EMOJIS. Name and
// roles are optional fields (though you'd want to change either though).
//
// Fires a Guild Emojis Update Gateway event.
func (c *Client) ModifyEmoji(guildID, emojiID discord.Snowflake, data ModifyEmojiData) error {
	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String(),
		httputil.WithJSONBody(data),
	)
}

// Delete the given emoji.
//
// Requires the MANAGE_EMOJIS permission.
// Fires a Guild Emojis Update Gateway event.
func (c *Client) DeleteEmoji(guildID, emojiID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String())
}
