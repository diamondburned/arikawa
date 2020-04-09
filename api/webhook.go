package api

import (
	"mime/multipart"
	"net/url"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/pkg/errors"
)

const EndpointWebhooks = Endpoint + "webhooks/"

// CreateWebhook creates a new webhook; avatar hash is optional. Requires
// MANAGE_WEBHOOKS.
func (c *Client) CreateWebhook(
	channelID discord.Snowflake,
	name string, avatar discord.Hash) (*discord.Webhook, error) {

	var param struct {
		Name   string       `json:"name"`
		Avatar discord.Hash `json:"avatar"`
	}

	param.Name = name
	param.Avatar = avatar

	var w *discord.Webhook
	return w, c.RequestJSON(
		&w, "POST",
		EndpointChannels+channelID.String()+"/webhooks",
		httputil.WithJSONBody(c, param),
	)
}

// Webhooks requires MANAGE_WEBHOOKS.
func (c *Client) Webhooks(
	guildID discord.Snowflake) ([]discord.Webhook, error) {

	var ws []discord.Webhook
	return ws, c.RequestJSON(&ws, "GET",
		EndpointGuilds+guildID.String()+"/webhooks")
}

func (c *Client) Webhook(
	webhookID discord.Snowflake) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET",
		EndpointWebhooks+webhookID.String())
}

func (c *Client) WebhookWithToken(
	webhookID discord.Snowflake, token string) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET",
		EndpointWebhooks+webhookID.String()+"/"+token)
}

type ModifyWebhookData struct {
	Name      string            `json:"name,omitempty"`
	Avatar    discord.Hash      `json:"avatar,omitempty"` // TODO: clear avatar how?
	ChannelID discord.Snowflake `json:"channel_id,omitempty"`
}

func (c *Client) ModifyWebhook(
	webhookID discord.Snowflake,
	data ModifyWebhookData) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "PATCH",
		EndpointWebhooks+webhookID.String())
}

func (c *Client) ModifyWebhookWithToken(
	webhookID discord.Snowflake,
	data ModifyWebhookData, token string) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "PATCH",
		EndpointWebhooks+webhookID.String()+"/"+token)
}

func (c *Client) DeleteWebhook(webhookID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointWebhooks+webhookID.String())
}

func (c *Client) DeleteWebhookWithToken(
	webhookID discord.Snowflake, token string) error {

	return c.FastRequest("DELETE",
		EndpointWebhooks+webhookID.String()+"/"+token)
}

// ExecuteWebhook sends a message to the webhook. If wait is bool, Discord will
// wait for the message to be delivered and will return the message body. This
// also means the returned message will only be there if wait is true.
func (c *Client) ExecuteWebhook(
	webhookID discord.Snowflake, token string, wait bool,
	data ExecuteWebhookData) (*discord.Message, error) {

	for i, embed := range data.Embeds {
		if err := embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "Embed error at "+strconv.Itoa(i))
		}
	}

	var param = url.Values{}
	if wait {
		param.Set("wait", "true")
	}

	var URL = EndpointWebhooks + webhookID.String() + "/" + token +
		"?" + param.Encode()
	var msg *discord.Message

	if len(data.Files) == 0 {
		// No files, so no need for streaming.
		return msg, c.RequestJSON(&msg, "POST", URL,
			httputil.WithJSONBody(c, data))
	}

	writer := func(mw *multipart.Writer) error {
		return data.WriteMultipart(c, mw)
	}

	resp, err := c.MeanwhileMultipart(writer, "POST", URL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if !wait {
		// Since we didn't tell Discord to wait, we have nothing to parse.
		return nil, nil
	}

	return msg, c.DecodeStream(resp.Body, &msg)
}
