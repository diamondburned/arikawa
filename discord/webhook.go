package discord

import "time"

type Webhook struct {
	ID   WebhookID   `json:"id"`
	Type WebhookType `json:"type"`
	User User        `json:"user"` // creator

	GuildID   GuildID   `json:"guild_id,omitempty"`
	ChannelID ChannelID `json:"channel_id"`

	Name   string `json:"name"`
	Avatar Hash   `json:"avatar"`
	Token  string `json:"token"` // incoming webhooks only
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
