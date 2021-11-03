package gateway

import (
	"context"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// DefaultPresence is used as the default presence when initializing a new
// Gateway.
var DefaultPresence *UpdateStatusData

// Identifier is a wrapper around IdentifyData to add in appropriate rate
// limiters.
type Identifier struct {
	IdentifyData

	IdentifyShortLimit  *rate.Limiter `json:"-"` // optional
	IdentifyGlobalLimit *rate.Limiter `json:"-"` // optional
}

// DefaultIdentifier creates a new default Identifier
func DefaultIdentifier(token string) *Identifier {
	return NewIdentifier(DefaultIdentifyData(token))
}

// NewIdentifier creates a new identifier with the given IdentifyData and
// default rate limiters.
func NewIdentifier(data IdentifyData) *Identifier {
	return &Identifier{
		IdentifyData:        data,
		IdentifyShortLimit:  rate.NewLimiter(rate.Every(5*time.Second), 1),
		IdentifyGlobalLimit: rate.NewLimiter(rate.Every(24*time.Hour), 1000),
	}
}

// Wait waits for the rate limiters to pass. If a limiter is nil, then it will
// not be used to wait. This is useful
func (i *Identifier) Wait(ctx context.Context) error {
	if i.IdentifyShortLimit != nil {
		if err := i.IdentifyShortLimit.Wait(ctx); err != nil {
			return errors.Wrap(err, "can't wait for short limit")
		}
	}

	if i.IdentifyGlobalLimit != nil {
		if err := i.IdentifyGlobalLimit.Wait(ctx); err != nil {
			return errors.Wrap(err, "can't wait for global limit")
		}
	}

	return nil
}

// DefaultIdentity is used as the default identity when initializing a new
// Gateway.
var DefaultIdentity = IdentifyProperties{
	OS:      runtime.GOOS,
	Browser: "Arikawa",
	Device:  "Arikawa",
}

// IdentifyData is the struct for a data that's sent over in an Identify
// command.
type IdentifyData struct {
	Token      string             `json:"token"`
	Properties IdentifyProperties `json:"properties"`

	Compress       bool `json:"compress,omitempty"`        // true
	LargeThreshold uint `json:"large_threshold,omitempty"` // 50

	Shard *Shard `json:"shard,omitempty"` // [ shard_id, num_shards ]

	Presence *UpdateStatusData `json:"presence,omitempty"`

	// ClientState is the client state for a user's accuont. Bot accounts should
	// NOT touch this field.
	ClientState *ClientState `json:"client_state,omitempty"`

	// Capabilities defines the client's capabilities when connecting to the
	// gateway with a user account. Bot accounts should NOT touch this field.
	// The official client sets this at 125 at the time of this commit.
	Capabilities int `json:"capabilities,omitempty"`
	// Intents specifies which groups of events the gateway
	// connection will receive.
	//
	// For user accounts, it must be nil.
	//
	// For bot accounts, it must not be nil, and
	// Gateway.AddIntents(0) can be used if you want to
	// specify no intents.
	Intents option.Uint `json:"intents"`
}

// DefaultIdentifyData creates a default IdentifyData with the given token.
func DefaultIdentifyData(token string) IdentifyData {
	return IdentifyData{
		Token:      token,
		Properties: DefaultIdentity,
		Presence:   DefaultPresence,

		Compress:       true,
		LargeThreshold: 50,
	}
}

// SetShard is a helper function to set the shard configuration inside
// IdentifyData.
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
	OSVersion        string `json:"os_version,omitempty"`
	Referrer         string `json:"referrer,omitempty"`
	ReferringDomain  string `json:"referring_domain,omitempty"`
}

// Shard is a type for two numbers that represent the Bot's shard configuration.
// The first number is the shard's ID, which could be obtained through the
// ShardID method. The second number is the total number of shards, which could
// be obtained through the NumShards method.
type Shard [2]int

// DefaultShard returns the default shard configuration of 1 shard total, in
// which the current shard ID is 0.
var DefaultShard = &Shard{0, 1}

// ShardID returns the current shard's ID. It uses the first number.
func (s Shard) ShardID() int {
	return s[0]
}

// NumShards returns the total number of shards. It uses the second number.
func (s Shard) NumShards() int {
	return s[1]
}

// ClientState describes the undocumented client_state field in the Identify
// command. Little is known about this type.
type ClientState struct {
	GuildHashes          map[discord.GuildID]interface{} `json:"guild_hashes"`            // {}
	HighestLastMessageID discord.MessageID               `json:"highest_last_message_id"` // "0"

	ReadStateVersion         int `json:"read_state_version"`          // 0
	UserGuildSettingsVersion int `json:"user_guild_settings_version"` // -1
	UserSettingsVersion      int `json:"user_settings_version"`       // -1
}
