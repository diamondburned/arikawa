package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

// Messages gets all mesesages, automatically paginating. Use with care, as
// this could get as many as hundred thousands of messages, making a lot of
// queries.
func (c *Client) Messages(channelID discord.Snowflake, max uint) ([]discord.Message, error) {
	var msgs []discord.Message
	var after discord.Snowflake = 0

	const hardLimit int = 100

	for fetch := uint(hardLimit); max > 0; fetch = uint(hardLimit) {
		if max > 0 {
			if fetch > max {
				fetch = max
			}
			max -= fetch
		}

		m, err := c.messagesRange(channelID, 0, after, 0, fetch)
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, m...)

		if len(m) < hardLimit {
			break
		}

		after = m[hardLimit-1].Author.ID
	}

	return msgs, nil
}

// MessagesAround returns messages around the ID, with a limit of 1-100.
func (c *Client) MessagesAround(
	channelID, around discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messagesRange(channelID, 0, 0, around, limit)
}

// MessagesBefore returns messages before the ID, with a limit of 1-100.
func (c *Client) MessagesBefore(
	channelID, before discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messagesRange(channelID, before, 0, 0, limit)
}

// MessagesAfter returns messages after the ID, with a limit of 1-100.
func (c *Client) MessagesAfter(
	channelID, after discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messagesRange(channelID, 0, after, 0, limit)
}

func (c *Client) messagesRange(
	channelID, before, after, around discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	switch {
	case limit == 0:
		limit = 50
	case limit > 100:
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before,omitempty"`
		After  discord.Snowflake `schema:"after,omitempty"`
		Around discord.Snowflake `schema:"around,omitempty"`

		Limit uint `schema:"limit"`
	}

	param.Before = before
	param.After = after
	param.Around = around
	param.Limit = limit

	var msgs []discord.Message
	return msgs, c.RequestJSON(
		&msgs, "GET",
		EndpointChannels+channelID.String()+"/messages",
		httputil.WithSchema(c, param),
	)
}

func (c *Client) Message(channelID, messageID discord.Snowflake) (*discord.Message, error) {
	var msg *discord.Message
	return msg, c.RequestJSON(&msg, "GET",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String())
}

func (c *Client) SendMessage(
	channelID discord.Snowflake, content string,
	embed *discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Content: content,
		Embed:   embed,
	})
}

func (c *Client) EditMessage(
	channelID, messageID discord.Snowflake, content string,
	embed *discord.Embed, suppressEmbeds bool) (*discord.Message, error) {

	var param struct {
		Content string               `json:"content,omitempty"`
		Embed   *discord.Embed       `json:"embed,omitempty"`
		Flags   discord.MessageFlags `json:"flags,omitempty"`
	}

	param.Content = content
	param.Embed = embed
	if suppressEmbeds {
		param.Flags = discord.SuppressEmbeds
	}

	var msg *discord.Message
	return msg, c.RequestJSON(
		&msg, "PATCH",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String(),
		httputil.WithJSONBody(param),
	)
}

// DeleteMessage deletes a message. Requires MANAGE_MESSAGES if the message is
// not made by yourself.
func (c *Client) DeleteMessage(channelID, messageID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String()+
		"/messages/"+messageID.String())
}

// DeleteMessages only works for bots. It can't delete messages older than 2
// weeks, and will fail if tried. This endpoint requires MANAGE_MESSAGES.
func (c *Client) DeleteMessages(channelID discord.Snowflake, messageIDs []discord.Snowflake) error {
	var param struct {
		Messages []discord.Snowflake `json:"messages"`
	}

	param.Messages = messageIDs

	return c.FastRequest("POST", EndpointChannels+channelID.String()+
		"/messages/bulk-delete", httputil.WithJSONBody(param))
}
