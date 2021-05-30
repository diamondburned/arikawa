package discord

import "time"

type Webhook struct {
	Avatar    Hash        `json:"avatar"`
	Token     string      `json:"token"` // incoming webhooks only
	Name      string      `json:"name"`
	User      User        `json:"user"` // creator
	GuildID   GuildID     `json:"guild_id,omitempty"`
	ChannelID ChannelID   `json:"channel_id"`
	ID        WebhookID   `json:"id"`
	Type      WebhookType `json:"type"`
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
