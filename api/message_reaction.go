package api

import (
	"net/url"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

// React creates a reaction for the message.
//
// This endpoint requires the READ_MESSAGE_HISTORY permission to be present on
// the current user. Additionally, if nobody else has reacted to the message
// using this emoji, this endpoint requires the 'ADD_REACTIONS' permission to
// be present on the current user.
func (c *Client) React(channelID, messageID discord.Snowflake, emoji Emoji) error {
	var msgURL = EndpointChannels + channelID.String() +
		"/messages/" + messageID.String() +
		"/reactions/" + url.PathEscape(emoji) + "/@me"
	return c.FastRequest("PUT", msgURL)
}

// Unreact removes a reaction the current user has made for the message.
func (c *Client) Unreact(chID, msgID discord.Snowflake, emoji Emoji) error {
	return c.DeleteUserReaction(chID, msgID, 0, emoji)
}

// Reactions returns a list of users that reacted with the passed Emoji. This
// method automatically paginates until it reaches the passed limit, or, if the
// limit is set to 0, has fetched all users within the passed range.
//
// As the underlying endpoint has a maximum of 100 users per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
//
// When fetching the users, those with the smallest ID will be fetched first.
func (c *Client) Reactions(
	channelID, messageID discord.Snowflake, limit uint, emoji Emoji) ([]discord.User, error) {

	return c.ReactionsAfter(channelID, messageID, 0, limit, emoji)
}

// ReactionsBefore returns a list of users that reacted with the passed Emoji.
// This method automatically paginates until it reaches the passed limit, or,
// if the limit is set to 0, has fetched all users within the passed range.
//
// As the underlying endpoint has a maximum of 100 users per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
func (c *Client) ReactionsBefore(
	channelID, messageID, before discord.Snowflake,
	limit uint, emoji Emoji) ([]discord.User, error) {

	var users []discord.User

	const hardLimit int = 100

	unlimited := limit == 0

	for fetch := uint(hardLimit); limit > 0 || unlimited; fetch = uint(hardLimit) {
		if limit > 0 {
			if fetch > limit {
				fetch = limit
			}
			limit -= fetch
		}

		r, err := c.reactionsRange(channelID, messageID, before, 0, fetch, emoji)
		if err != nil {
			return users, err
		}
		users = append(users, r...)

		if len(r) < hardLimit {
			break
		}

		before = r[0].ID
	}

	return users, nil
}

// ReactionsAfter returns a list of users that reacted with the passed Emoji.
// This method automatically paginates until it reaches the passed limit, or,
// if the limit is set to 0, has fetched all users within the passed range.
//
// As the underlying endpoint has a maximum of 100 users per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
func (c *Client) ReactionsAfter(
	channelID, messageID, after discord.Snowflake, limit uint, emoji Emoji,
) ([]discord.User, error) {

	var users []discord.User

	const hardLimit int = 100

	unlimited := limit == 0

	for fetch := uint(hardLimit); limit > 0 || unlimited; fetch = uint(hardLimit) {
		if limit > 0 {
			if fetch > limit {
				fetch = limit
			}
			limit -= fetch
		}

		r, err := c.reactionsRange(channelID, messageID, 0, after, fetch, emoji)
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

// reactionsRange get users before and after IDs. Before, after, and limit are
// optional. A maximum limit of only 100 reactions could be returned.
func (c *Client) reactionsRange(
	channelID, messageID, before, after discord.Snowflake,
	limit uint, emoji Emoji) ([]discord.User, error) {

	switch {
	case limit == 0:
		limit = 25
	case limit > 100:
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

// DeleteReaction deletes another user's reaction.
//
// This endpoint requires the MANAGE_MESSAGES permission to be present on the
// current user.
func (c *Client) DeleteUserReaction(
	channelID, messageID, userID discord.Snowflake, emoji Emoji) error {

	var user = "@me"
	if userID > 0 {
		user = userID.String()
	}

	return c.FastRequest(
		"DELETE",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+
			"/reactions/"+url.PathEscape(emoji)+"/"+user,
	)
}

// DeleteReactions deletes all the reactions for a given emoji on a message.
//
// This endpoint requires the MANAGE_MESSAGES permission to be present on the
// current user.
// Fires a Message Reaction Remove Emoji Gateway event.
func (c *Client) DeleteReactions(
	channelId, messageID discord.Snowflake, emoji Emoji) error {

	return c.FastRequest(
		"DELETE",
		EndpointChannels+channelId.String()+"/messages/"+messageID.String()+
			"/reactions/"+url.PathEscape(emoji),
	)
}

// DeleteAllReactions deletes all reactions on a message.
//
// This endpoint requires the MANAGE_MESSAGES permission to be present on the
// current user.
// Fires a Message Reaction Remove All Gateway event.
func (c *Client) DeleteAllReactions(channelID, messageID discord.Snowflake) error {
	return c.FastRequest(
		"DELETE",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+"/reactions/",
	)
}
