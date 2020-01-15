package gateway

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/pkg/errors"
)

// Rules: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent

type IdentifyData struct {
	Token      string             `json:"token"`
	Properties IdentifyProperties `json:"properties"`

	Compress          bool `json:"compress,omitempty"`        // true
	LargeThreshold    uint `json:"large_threshold,omitempty"` // 50
	GuildSubscription bool `json:"guild_subscriptions"`       // true

	Shard *Shard `json:"shard,omitempty"` // [ shard_id, num_shards ]

	Presence *UpdateStatusData `json:"presence,omitempty"`
}

func (i *IdentifyData) SetShard(id, num int) {
	if i.Shard == nil {
		i.Shard = new(Shard)
	}
	i.Shard[0], i.Shard[1] = id, num
}

type IdentifyProperties struct {
	// Required
	OS      string `json:"os"`      // GOOS
	Browser string `json:"browser"` // Arikawa
	Device  string `json:"device"`  // Arikawa

	// Optional
	BrowserUserAgent string `json:"browser_user_agent,omitempty"`
	BrowserVersion   string `json:"browser_version,omitempty"`
	OsVersion        string `json:"os_version,omitempty"`
	Referrer         string `json:"referrer,omitempty"`
	ReferringDomain  string `json:"referring_domain,omitempty"`
}

func (g *Gateway) Identify() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	if err := g.Identifier.Wait(ctx); err != nil {
		return errors.Wrap(err, "Can't wait for identify()")
	}

	return g.Send(IdentifyOP, g.Identifier)
}

type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// HeartbeatData is the last sequence number to be sent.
type HeartbeatData int

func (g *Gateway) Heartbeat() error {
	return g.Send(HeartbeatOP, g.Sequence.Get())
}

type RequestGuildMembersData struct {
	GuildID []discord.Snowflake `json:"guild_id"`
	UserIDs []discord.Snowflake `json:"user_id,omitempty"`

	Query     string `json:"query,omitempty"`
	Limit     uint   `json:"limit"`
	Presences bool   `json:"presences,omitempty"`
}

type UpdateVoiceStateData struct {
	GuildID   discord.Snowflake `json:"guild_id"`
	ChannelID discord.Snowflake `json:"channel_id"`
	SelfMute  bool              `json:"self_mute"`
	SelfDeaf  bool              `json:"self_deaf"`
}

type UpdateStatusData struct {
	Since discord.Milliseconds `json:"since,omitempty"` // 0 if not idle
	Game  *Activity            `json:"game,omitempty"`  // nullable

	Status Status `json:"status"`
	AFK    bool   `json:"afk"`
}
