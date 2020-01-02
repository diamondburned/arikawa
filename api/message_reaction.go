package api

import (
	"git.sr.ht/~diamondburned/arikawa/discord"
	"git.sr.ht/~diamondburned/arikawa/httputil"
)

// React adds a reaction to the message. This requires READ_MESSAGE_HISTORY (and
// additionally ADD_REACTIONS) to react.
func (c *Client) React(chID, msgID discord.Snowflake,
	emoji EmojiAPI) error {

	var msgURL = EndpointChannels + chID.String() +
		"/messages/" + msgID.String() +
		"/reactions/" + emoji + "/@me"
	return c.FastRequest("PUT", msgURL)
}

func (c *Client) Reactions(chID, msgID discord.Snowflake,
	limit uint, emoji EmojiAPI) ([]discord.User, error) {

	return c.ReactionRange(chID, msgID, 0, 0, limit, emoji)
}

// ReactionRange get users before and after IDs. Before, after, and limit are
// optional.
func (c *Client) ReactionRange(
	chID, msgID, before, after discord.Snowflake,
	limit uint, emoji EmojiAPI) ([]discord.User, error) {

	if limit == 0 {
		limit = 25
	}

	if limit > 100 {
		limit = 100
	}

	var query struct {
		Before discord.Snowflake `json:"before,omitempty"`
		After  discord.Snowflake `json:"after,omitempty"`

		Limit uint `json:"limit"`
	}

	var users []discord.User
	var msgURL = EndpointChannels + chID.String() +
		"/messages/" + msgID.String() +
		"/reactions/" + emoji

	return users, c.RequestJSON(&users, "GET", msgURL,
		httputil.WithJSONBody(c, query))
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
