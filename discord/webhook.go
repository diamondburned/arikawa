package discord

import "time"

// Webhook is used to represent a webhook.
//
// https://discord.com/developers/docs/resources/webhook#webhook-object
type Webhook struct {
	// ID is the id of the webhook.
	ID WebhookID `json:"id"`
	// Type is the WebhookType of the webhook.
	Type WebhookType `json:"type"`
	// GuildID is the guild id this webhook is for, if any.
	GuildID GuildID `json:"guild_id,omitempty"`
	// ChannelID is the channel id this webhook is for, if any.
	ChannelID ChannelID `json:"channel_id"`
	// User is the user this webhook was created by.
	//
	// This field is not returned when getting a webhook with its token.
	User *User `json:"user,omitempty"`

	// Name is the default name of the webhook.
	Name string `json:"name"`
	// Avatar is the default user avatar hash of the webhook.
	Avatar Hash `json:"avatar"`
	// Token is the secure token of the webhook, returned for incoming
	// webhooks.
	Token string `json:"token,omitempty"`

	// ApplicationID is the bot/OAuth2 application that created this webhook.
	ApplicationID AppID `json:"application_id"`

	// SourceGuild is the guild of the channel that this webhook is following.
	// It is returned for channel follower webhooks.
	//
	// This field will only be filled partially.
	SourceGuild *Guild `json:"source_guild,omitempty"`
	// SourceChannel is the channel that this webhook is following. It is
	// returned for channel follower webhooks.
	//
	// This field will only be filled partially.
	SourceChannel *Channel `json:"source_channel,omitempty"`
	// URL is the url used for executing the webhook. It is returned by the
	// webhooks OAuth2 flow.
	URL URL `json:"url,omitempty"`
}

// CreatedAt returns a time object representing when the webhook was created.
func (w Webhook) CreatedAt() time.Time {
	return w.ID.Time()
}

type WebhookType uint8

const (
	_ WebhookType = iota
	IncomingWebhook
	ChannelFollowerWebhook
)
