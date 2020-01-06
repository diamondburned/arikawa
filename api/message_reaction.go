package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/httputil"
)

// React adds a reaction to the message. This requires READ_MESSAGE_HISTORY (and
// additionally ADD_REACTIONS) to react.
func (c *Client) React(
	channelID, messageID discord.Snowflake, emoji EmojiAPI) error {

	var msgURL = EndpointChannels + channelID.String() +
		"/messages/" + messageID.String() +
		"/reactions/" + emoji + "/@me"
	return c.FastRequest("PUT", msgURL)
}

// Reactions returns all reactions. It will paginate automatically.
func (c *Client) Reactions(
	channelID, messageID discord.Snowflake,
	emoji EmojiAPI) ([]discord.User, error) {

	var users []discord.User
	var after discord.Snowflake = 0

	for {
		r, err := c.ReactionsRange(channelID, messageID, 0, after, 100, emoji)
		if err != nil {
			return users, err
		}
		users = append(users, r...)

		if len(r) < 100 {
			break
		}

		after = r[99].ID
	}

	return users, nil
}

// Refer to ReactionsRange.
func (c *Client) ReactionsBefore(
	channelID, messageID, before discord.Snowflake,
	limit uint, emoji EmojiAPI) ([]discord.User, error) {

	return c.ReactionsRange(channelID, messageID, before, 0, limit, emoji)
}

// Refer to ReactionsRange.
func (c *Client) ReactionsAfter(
	channelID, messageID, after discord.Snowflake,
	limit uint, emoji EmojiAPI) ([]discord.User, error) {

	return c.ReactionsRange(channelID, messageID, 0, after, limit, emoji)
}

// ReactionsRange get users before and after IDs. Before, after, and limit are
// optional. A maximum limit of only 100 reactions could be returned.
func (c *Client) ReactionsRange(
	channelID, messageID, before, after discord.Snowflake,
	limit uint, emoji EmojiAPI) ([]discord.User, error) {

	if limit == 0 {
		limit = 25
	}

	if limit > 100 {
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before,omitempty"`
		After  discord.Snowflake `schema:"after,omitempty"`

		Limit uint `schema:"limit"`
	}

	param.Before = before
	param.After = after
	param.Limit = limit

	var users []discord.User
	return users, c.RequestJSON(
		&users, "GET", EndpointChannels+channelID.String()+
			"/messages/"+messageID.String()+
			"/reactions/"+emoji,
		httputil.WithSchema(c, param),
	)
}

// DeleteReaction requires MANAGE_MESSAGES if not @me.
func (c *Client) DeleteReaction(
	chID, msgID, userID discord.Snowflake, emoji EmojiAPI) error {

	var user = "@me"
	if userID > 0 {
		user = userID.String()
	}

	var msgURL = EndpointChannels + chID.String() +
		"/messages/" + msgID.String() +
		"/reactions/" + emoji + "/" + user

	return c.FastRequest("DELETE", msgURL)
}

func (c *Client) DeleteOwnReaction(
	chID, msgID discord.Snowflake, emoji EmojiAPI) error {

	return c.DeleteReaction(chID, msgID, 0, emoji)
}

// DeleteAllReactions equires MANAGE_MESSAGE.
func (c *Client) DeleteAllReactions(
	chID, msgID discord.Snowflake, emoji EmojiAPI) error {

	var msgURL = EndpointChannels + chID.String() +
		"/messages/" + msgID.String() +
		"/reactions/" + emoji

	return c.FastRequest("DELETE", msgURL)
}
