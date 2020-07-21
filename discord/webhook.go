package discord

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

type WebhookType uint8

const (
	_ WebhookType = iota
	IncomingWebhook
	ChannelFollowerWebhook
)
