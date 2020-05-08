package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

var EndpointWebhooks = Endpoint + "webhooks/"

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
		httputil.WithJSONBody(param),
	)
}

// Webhooks requires MANAGE_WEBHOOKS.
func (c *Client) Webhooks(guildID discord.Snowflake) ([]discord.Webhook, error) {
	var ws []discord.Webhook
	return ws, c.RequestJSON(&ws, "GET", EndpointGuilds+guildID.String()+"/webhooks")
}

func (c *Client) Webhook(webhookID discord.Snowflake) (*discord.Webhook, error) {
	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET", EndpointWebhooks+webhookID.String())
}

func (c *Client) WebhookWithToken(
	webhookID discord.Snowflake, token string) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET", EndpointWebhooks+webhookID.String()+"/"+token)
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
	return w, c.RequestJSON(&w, "PATCH", EndpointWebhooks+webhookID.String())
}

func (c *Client) ModifyWebhookWithToken(
	webhookID discord.Snowflake,
	data ModifyWebhookData, token string) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(&w, "PATCH", EndpointWebhooks+webhookID.String()+"/"+token)
}

func (c *Client) DeleteWebhook(webhookID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointWebhooks+webhookID.String())
}

func (c *Client) DeleteWebhookWithToken(webhookID discord.Snowflake, token string) error {
	return c.FastRequest("DELETE", EndpointWebhooks+webhookID.String()+"/"+token)
}
