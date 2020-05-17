package gateway

import (
	"context"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// Identity is used as the default identity when initializing a new Gateway.
var Identity = IdentifyProperties{
	OS:      runtime.GOOS,
	Browser: "Arikawa",
	Device:  "Arikawa",
}

// Presence is used as the default presence when initializing a new Gateway.
var Presence *UpdateStatusData

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

type IdentifyData struct {
	Token      string             `json:"token"`
	Properties IdentifyProperties `json:"properties"`

	Compress           bool `json:"compress,omitempty"`        // true
	LargeThreshold     uint `json:"large_threshold,omitempty"` // 50
	GuildSubscriptions bool `json:"guild_subscriptions"`       // true

	Shard *Shard `json:"shard,omitempty"` // [ shard_id, num_shards ]

	Presence *UpdateStatusData `json:"presence,omitempty"`

	Intents Intents `json:"intents,omitempty"`
}

func (i *IdentifyData) SetShard(id, num int) {
	if i.Shard == nil {
		i.Shard = new(Shard)
	}
	i.Shard[0], i.Shard[1] = id, num
}

// Intents is a new Discord API feature that's documented at
// https://discordapp.com/developers/docs/topics/gateway#gateway-intents.
type Intents uint32

const (
	IntentGuilds Intents = 1 << iota
	IntentGuildMembers
	IntentGuildBans
	IntentGuildEmojis
	IntentGuildIntegrations
	IntentGuildWebhooks
	IntentGuildInvites
	IntentGuildVoiceStates
	IntentGuildPresences
	IntentGuildMessages
	IntentGuildMessageReactions
	IntentGuildMessageTyping
	IntentDirectMessages
	IntentDirectMessageReactions
	IntentDirectMessageTyping
)

type Identifier struct {
	IdentifyData

	IdentifyShortLimit  *rate.Limiter `json:"-"`
	IdentifyGlobalLimit *rate.Limiter `json:"-"`
}

func DefaultIdentifier(token string) *Identifier {
	return NewIdentifier(IdentifyData{
		Token:      token,
		Properties: Identity,
		Shard:      DefaultShard(),
		Presence:   Presence,

		Compress:           true,
		LargeThreshold:     50,
		GuildSubscriptions: true,
	})
}

func NewIdentifier(data IdentifyData) *Identifier {
	return &Identifier{
		IdentifyData:        data,
		IdentifyShortLimit:  rate.NewLimiter(rate.Every(5*time.Second), 1),
		IdentifyGlobalLimit: rate.NewLimiter(rate.Every(24*time.Hour), 1000),
	}
}

func (i *Identifier) Wait(ctx context.Context) error {
	if err := i.IdentifyShortLimit.Wait(ctx); err != nil {
		return errors.Wrap(err, "can't wait for short limit")
	}
	if err := i.IdentifyGlobalLimit.Wait(ctx); err != nil {
		return errors.Wrap(err, "can't wait for global limit")
	}
	return nil
}
