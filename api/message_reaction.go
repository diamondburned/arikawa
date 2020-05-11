package api

import (
	"net/url"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

// React adds a reaction to the message. This requires READ_MESSAGE_HISTORY (and
// additionally ADD_REACTIONS) to react.
func (c *Client) React(
	channelID, messageID discord.Snowflake, emoji Emoji) error {

	var msgURL = EndpointChannels + channelID.String() +
		"/messages/" + messageID.String() +
		"/reactions/" + url.PathEscape(emoji) + "/@me"
	return c.FastRequest("PUT", msgURL)
}

// Unreact removes own's reaction from the message.
func (c *Client) Unreact(chID, msgID discord.Snowflake, emoji Emoji) error {
	return c.DeleteUserReaction(chID, msgID, 0, emoji)
}

// Reactions returns all reactions. It will paginate automatically.
func (c *Client) Reactions(
	channelID, messageID discord.Snowflake,
	max uint, emoji Emoji) ([]discord.User, error) {

	var users []discord.User
	var after discord.Snowflake = 0

	const hardLimit int = 100

	for fetch := uint(hardLimit); max > 0; fetch = uint(hardLimit) {
		if max > 0 {
			if fetch > max {
				fetch = max
			}
			max -= fetch
		}

		r, err := c.ReactionsRange(channelID, messageID, 0, after, fetch, emoji)
		if err != nil {
			return users, err
		}
		users = append(users, r...)

		if len(r) < hardLimit {
			break
		}

		after = r[hardLimit-1].ID
	}

	return users, nil
}

// Refer to ReactionsRange.
func (c *Client) ReactionsBefore(
	channelID, messageID, before discord.Snowflake,
	limit uint, emoji Emoji) ([]discord.User, error) {

	return c.ReactionsRange(channelID, messageID, before, 0, limit, emoji)
}

// Refer to ReactionsRange.
func (c *Client) ReactionsAfter(
	channelID, messageID, after discord.Snowflake,
	limit uint, emoji Emoji) ([]discord.User, error) {

	return c.ReactionsRange(channelID, messageID, 0, after, limit, emoji)
}

// ReactionsRange get users before and after IDs. Before, after, and limit are
// optional. A maximum limit of only 100 reactions could be returned.
func (c *Client) ReactionsRange(
	channelID, messageID, before, after discord.Snowflake,
	limit uint, emoji Emoji) ([]discord.User, error) {

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
			"/reactions/"+url.PathEscape(emoji),
		httputil.WithSchema(c, param),
	)
}

// DeleteReaction requires MANAGE_MESSAGES if not @me.
func (c *Client) DeleteUserReaction(
	chID, msgID, userID discord.Snowflake, emoji Emoji) error {

	var user = "@me"
	if userID > 0 {
		user = userID.String()
	}

	return c.FastRequest("DELETE", EndpointChannels+chID.String()+
		"/messages/"+msgID.String()+
		"/reactions/"+url.PathEscape(emoji)+"/"+user)
}

// DeleteReactions equires MANAGE_MESSAGE.
func (c *Client) DeleteReactions(
	chID, msgID discord.Snowflake, emoji Emoji) error {

	return c.FastRequest("DELETE", EndpointChannels+chID.String()+
		"/messages/"+msgID.String()+
		"/reactions/"+url.PathEscape(emoji))
}

// DeleteAllReactions equires MANAGE_MESSAGE.
func (c *Client) DeleteAllReactions(chID, msgID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointChannels+chID.String()+
		"/messages/"+msgID.String()+"/reactions/")
}
