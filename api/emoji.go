package api

import (
	"errors"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/internal/httputil"
)

// EmojiAPI is a special format that the API wants.
type EmojiAPI = string

func FormatEmojiAPI(id discord.Snowflake, name string) string {
	if id == 0 {
		return name
	}

	return id.String() + ":" + name
}

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

var (
	ErrEmojiTooLarge = errors.New("Emoji is larger than 256k")
)

// CreateEmoji creates a new emoji in the guild. This endpoint requires
// MANAGE_EMOJIS. ContentType must be "image/jpeg", "image/png", or
// "image/gif". However, ContentType can also be automatically detected
// (though shouldn't be relied on). Roles slice is optional.
func (c *Client) CreateEmoji(
	guildID discord.Snowflake, name string, image Image,
	roles []discord.Snowflake) (*discord.Emoji, error) {

	image.MaxSize = 256 * 1000
	if err := image.Validate(); err != nil {
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
		httputil.WithJSONBody(c, param),
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

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String(),
		httputil.WithJSONBody(c, param),
	)
}

// DeleteEmoji requires MANAGE_EMOJIS.
func (c *Client) DeleteEmoji(guildID, emojiID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/emojis/"+emojiID.String())
}
