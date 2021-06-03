package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var EndpointWebhooks = Endpoint + "webhooks/"

// https://discord.com/developers/docs/resources/webhook#create-webhook-json-params
type CreateWebhookData struct {
	// Name is the name of the webhook (1-80 characters).
	Name string `json:"name"`
	// Avatar is the image for the default webhook avatar.
	Avatar *Image `json:"avatar"`
}

// CreateWebhook creates a new webhook.
//
// Webhooks cannot be named "clyde".
//
// Requires the MANAGE_WEBHOOKS permission.
func (c *Client) CreateWebhook(
	channelID discord.ChannelID, data CreateWebhookData) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(
		&w, "POST",
		EndpointChannels+channelID.String()+"/webhooks",
		httputil.WithJSONBody(data),
	)
}

// ChannelWebhooks returns the webhooks of the channel with the given ID.
//
// Requires the MANAGE_WEBHOOKS permission.
func (c *Client) ChannelWebhooks(channelID discord.ChannelID) ([]discord.Webhook, error) {
	var ws []discord.Webhook
	return ws, c.RequestJSON(&ws, "GET", EndpointChannels+channelID.String()+"/webhooks")
}

// GuildWebhooks returns the webhooks of the guild with the given ID.
//
// Requires the MANAGE_WEBHOOKS permission.
func (c *Client) GuildWebhooks(guildID discord.GuildID) ([]discord.Webhook, error) {
	var ws []discord.Webhook
	return ws, c.RequestJSON(&ws, "GET", EndpointGuilds+guildID.String()+"/webhooks")
}

// Webhook returns the webhook with the given id.
func (c *Client) Webhook(webhookID discord.WebhookID) (*discord.Webhook, error) {
	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET", EndpointWebhooks+webhookID.String())
}

// https://discord.com/developers/docs/resources/webhook#modify-webhook-json-params
type ModifyWebhookData struct {
	// Name is the default name of the webhook.
	Name option.String `json:"name,omitempty"`
	// Avatar is the image for the default webhook avatar.
	Avatar *Image `json:"avatar,omitempty"`
	// ChannelID is the new channel id this webhook should be moved to.
	ChannelID discord.ChannelID `json:"channel_id,omitempty"`
}

// ModifyWebhook modifies a webhook.
//
// Requires the MANAGE_WEBHOOKS permission.
func (c *Client) ModifyWebhook(
	webhookID discord.WebhookID, data ModifyWebhookData) (*discord.Webhook, error) {

	var w *discord.Webhook
	return w, c.RequestJSON(
		&w, "PATCH",
		EndpointWebhooks+webhookID.String(),
		httputil.WithJSONBody(data),
	)
}

// DeleteWebhook deletes a webhook permanently.
//
// Requires the MANAGE_WEBHOOKS permission.
func (c *Client) DeleteWebhook(webhookID discord.WebhookID) error {
	return c.FastRequest("DELETE", EndpointWebhooks+webhookID.String())
}
