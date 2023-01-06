package discord

type VoiceState struct {
	// GuildID isn't available from the Guild struct.
	GuildID GuildID `json:"guild_id"`

	ChannelID ChannelID `json:"channel_id"`
	UserID    UserID    `json:"user_id"`
	Member    *Member   `json:"member,omitempty"`
	SessionID string    `json:"session_id"`

	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`

	SelfDeaf   bool `json:"self_deaf"`
	SelfMute   bool `json:"self_mute"`
	SelfStream bool `json:"self_stream,omitempty"`
	SelfVideo  bool `json:"self_video,omitempty"`
	Suppress   bool `json:"suppress"`

	RequestToSpeakTimestamp *Timestamp `json:"request_to_speak_timestamp"`
}

type VoiceRegion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	VIP        bool   `json:"vip"`
	Optimal    bool   `json:"optimal"`
	Deprecated bool   `json:"deprecated"`
	Custom     bool   `json:"custom"` // used for events
}
