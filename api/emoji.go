package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

// Emojis returns a list of emoji objects for the given guild.
func (c *Client) Emojis(guildID discord.GuildID) ([]discord.Emoji, error) {
	var e []discord.Emoji
	return e, c.RequestJSON(&e, "GET", EndpointGuilds+guildID.String()+"/emojis")
}

// Emoji returns an emoji object for the given guild and emoji IDs.
func (c *Client) Emoji(guildID discord.GuildID, emojiID discord.EmojiID) (*discord.Emoji, error) {
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
	// Roles are the roles that can use the emoji.
	Roles *[]discord.RoleID `json:"roles,omitempty"`

	AuditLogReason `json:"-"`
}

// CreateEmoji creates a new emoji in the guild. This endpoint requires
// MANAGE_EMOJIS. ContentType must be "image/jpeg", "image/png", or
// "image/gif". However, ContentType can also be automatically detected (though
// shouldn't be relied on).
//
// Emojis and animated emojis have a maximum file size of 256kb.
func (c *Client) CreateEmoji(
	guildID discord.GuildID, data CreateEmojiData) (*discord.Emoji, error) {

	// Max 256KB
	if err := data.Image.Validate(256 * 1000); err != nil {
		return nil, err
	}

	var emj *discord.Emoji
	return emj, c.RequestJSON(
		&emj, "POST",
		EndpointGuilds+guildID.String()+"/emojis",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// https://discord.com/developers/docs/resources/emoji#modify-guild-emoji-json-params
type ModifyEmojiData struct {
	// Name is the name of the emoji.
	Name string `json:"name,omitempty"`
	// Roles are the roles that can use the emoji.
	Roles *[]discord.RoleID `json:"roles,omitempty"`

	AuditLogReason `json:"-"`
}

// ModifyEmoji changes an existing emoji. This requires MANAGE_EMOJIS. Name and
// roles are optional fields (though you'd want to change either though).
//
// Fires a Guild Emojis Update Gateway event.
func (c *Client) ModifyEmoji(
	guildID discord.GuildID, emojiID discord.EmojiID, data ModifyEmojiData) error {

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// DeleteEmoji deletes the given emoji.
//
// Requires the MANAGE_EMOJIS permission.
//
// Fires a Guild Emojis Update Gateway event.
func (c *Client) DeleteEmoji(
	guildID discord.GuildID, emojiID discord.EmojiID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}
