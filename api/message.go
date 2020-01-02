package api

import (
	"io"

	"git.sr.ht/~diamondburned/arikawa/discord"
	"git.sr.ht/~diamondburned/arikawa/httputil"
	"github.com/pkg/errors"
)

func (c *Client) Messages(channelID discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messages(channelID, limit, nil)
}

func (c *Client) MessagesAround(channelID, around discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messages(channelID, limit, map[string]interface{}{
		"around": around,
	})
}

func (c *Client) MessagesBefore(channelID, before discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messages(channelID, limit, map[string]interface{}{
		"before": before,
	})
}

func (c *Client) MessagesAfter(channelID, after discord.Snowflake,
	limit uint) ([]discord.Message, error) {

	return c.messages(channelID, limit, map[string]interface{}{
		"after": after,
	})
}

func (c *Client) messages(channelID discord.Snowflake,
	limit uint, body map[string]interface{}) ([]discord.Message, error) {

	if body == nil {
		body = map[string]interface{}{}
	}

	switch {
	case limit == 0:
		limit = 50
	case limit > 100:
		limit = 100
	}

	body["limit"] = limit

	var msgs []discord.Message
	return msgs, c.RequestJSON(&msgs, "GET",
		EndpointChannels+channelID.String(), httputil.WithJSONBody(c, body))
}

func (c *Client) Message(
	channelID, messageID discord.Snowflake) (*discord.Message, error) {

	var msg *discord.Message
	return msg, c.RequestJSON(&msg, "GET",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String())
}

func (c *Client) SendMessage(channelID discord.Snowflake,
	content string, embed *discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Content: content,
		Embed:   embed,
	})
}

func (c *Client) SendMessageComplex(channelID discord.Snowflake,
	data SendMessageData) (*discord.Message, error) {

	if data.Embed != nil {
		if err := data.Embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "Embed error")
		}
	}

	var URL = EndpointChannels + channelID.String()
	var msg *discord.Message

	if len(data.Files) == 0 {
		// No files, no need for streaming
		return msg, c.RequestJSON(&msg, "POST", URL,
			httputil.WithJSONBody(c, data))
	}

	writer := func(w io.Writer) error {
		return data.WriteMultipart(c, w)
	}

	resp, err := c.MeanwhileBody(writer, "POST", URL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return msg, c.DecodeStream(resp.Body, &msg)
}

func (c *Client) EditMessage(channelID, messageID discord.Snowflake,
	content string, embed *discord.Embed, suppressEmbeds bool,
) (*discord.Message, error) {

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
		httputil.WithJSONBody(c, param),
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
func (c *Client) DeleteMessages(channelID discord.Snowflake,
	messageIDs []discord.Snowflake) error {

	var param struct {
		Messages []discord.Snowflake `json:"messages"`
	}

	param.Messages = messageIDs

	return c.FastRequest("POST", EndpointChannels+channelID.String()+
		"/messages/bulk-delete", httputil.WithJSONBody(c, param))
}
