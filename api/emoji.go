package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

// Emoji is a special format that the API wants.
type Emoji = string

// NewEmojiFromGuildEmoji creates a new Emoji using a custom guild emoji as
// base.
func NewEmojiFromGuildEmoji(id discord.Snowflake, name string) Emoji {
	return name + ":" + id.String()
}

// NewEmojiFromUnicode creates a new Emoji with the passed unicode emoji as
// base.
func NewEmojiFromUnicode(emoji string) Emoji { return emoji }

func (c *Client) Emojis(
	guildID discord.Snowflake) ([]discord.Emoji, error) {

	var emjs []discord.Emoji
	return emjs, c.RequestJSON(&emjs, "GET",
		EndpointGuilds+guildID.String()+"/emojis")
}

func (c *Client) Emoji(
	guildID, emojiID discord.Snowflake) (*discord.Emoji, error) {

	var emj *discord.Emoji
	return emj, c.RequestJSON(&emj, "GET",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String())
}

// CreateEmoji creates a new emoji in the guild. This endpoint requires
// MANAGE_EMOJIS. ContentType must be "image/jpeg", "image/png", or
// "image/gif". However, ContentType can also be automatically detected
// (though shouldn't be relied on). Roles slice is optional.
func (c *Client) CreateEmoji(
	guildID discord.Snowflake, name string, image Image,
	roles []discord.Snowflake) (*discord.Emoji, error) {

	// Max 256KB
	if err := image.Validate(256 * 1000); err != nil {
		return nil, err
	}

	var param struct {
		Name  string              `json:"name"`
		Image Image               `json:"image"`
		Roles []discord.Snowflake `json:"roles"`
	}

	param.Name = name
	param.Roles = roles
	param.Image = image

	var emj *discord.Emoji
	return emj, c.RequestJSON(
		&emj, "POST",
		EndpointGuilds+guildID.String()+"/emojis",
		httputil.WithJSONBody(param),
	)
}

// ModifyEmoji changes an existing emoji. This requires MANAGE_EMOJIS. Name and
// roles are optional fields (though you'd want to change either though).
func (c *Client) ModifyEmoji(
	guildID, emojiID discord.Snowflake, name string,
	roles []discord.Snowflake) error {

	var param struct {
		Name  string              `json:"name,omitempty"`
		Roles []discord.Snowflake `json:"roles,omitempty"`
	}

	param.Name = name
	param.Roles = roles

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String(),
		httputil.WithJSONBody(param),
	)
}

// DeleteEmoji requires MANAGE_EMOJIS.
func (c *Client) DeleteEmoji(guildID, emojiID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String())
}
