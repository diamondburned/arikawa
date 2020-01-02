package api

import (
	"git.sr.ht/~diamondburned/arikawa/discord"
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

// func (c *Client) CreateEmoji()
