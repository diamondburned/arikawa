package discord

type VoiceState struct {
	// GuildID isn't available from the Guild struct.
	GuildID Snowflake `json:"guild_id,string"`

	ChannelID Snowflake `json:"channel_id,string"`
	UserID    Snowflake `json:"user_id,string"`
	Member    *Member   `json:"member,omitempty"`
	SessionID string    `json:"session_id"`

	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`

	SelfDeaf   bool `json:"self_deaf"`
	SelfMute   bool `json:"self_mute"`
	SelfStream bool `json:"self_stream,omitempty"`
	Suppress   bool `json:"suppress"`
}

type VoiceRegion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	VIP        bool   `json:"vip"`
	Optimal    bool   `json:"optimal"`
	Deprecated bool   `json:"deprecated"`
	Custom     bool   `json:"custom"` // used for events
}
