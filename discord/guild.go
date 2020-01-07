package discord

type Guild struct {
	ID     Snowflake `json:"id,string"`
	Name   string    `json:"name"`
	Icon   Hash      `json:"icon"`
	Splash Hash      `json:"splash,omitempty"` // server invite bg

	Owner   bool      `json:"owner,omitempty"` // self is owner
	OwnerID Snowflake `json:"owner_id,string"`

	Permissions Permissions `json:"permissions,omitempty"`

	VoiceRegion string `json:"region"`

	AFKChannelID Snowflake `json:"afk_channel_id,string,omitempty"`
	AFKTimeout   Seconds   `json:"afk_timeout"`

	Embeddable     bool      `json:"embed_enabled,omitempty"`
	EmbedChannelID Snowflake `json:"embed_channel_id,string,omitempty"`

	Verification   Verification   `json:"verification_level"`
	Notification   Notification   `json:"default_message_notifications"`
	ExplicitFilter ExplicitFilter `json:"explicit_content_filter"`

	Roles    []Role         `json:"roles"`
	Emojis   []Emoji        `json:"emojis"`
	Features []GuildFeature `json:"guild_features"`

	MFA MFALevel `json:"mfa"`

	AppID Snowflake `json:"application_id,string,omitempty"`

	Widget bool `json:"widget_enabled,omitempty"`

	WidgetChannelID Snowflake `json:"widget_channel_id,string,omitempty"`
	SystemChannelID Snowflake `json:"system_channel_id,string,omitempty"`

	// GUILD_CREATE only.
	Joined      Timestamp    `json:"timestamp,omitempty"`
	Large       bool         `json:"large,omitempty"`
	Unavailable bool         `json:"unavailable,omitempty"`
	MemberCount uint64       `json:"member_count,omitempty"`
	VoiceStates []VoiceState `json:"voice_state,omitempty"`
	Members     []Member     `json:"members,omitempty"`
	Channels    []Channel    `json:"channel,omitempty"`
	Presences   []Presence   `json:"presences,omitempty"`

	// It's DefaultMaxPresences when MaxPresences is 0.
	MaxPresences uint64 `json:"max_presences,omitempty"`
	MaxMembers   uint64 `json:"max_members,omitempty"`

	VanityURLCode string `json:"vanity_url_code,omitempty"`
	Description   string `json:"description,omitempty"`
	Banner        Hash   `json:"banner,omitempty"`

	NitroBoost    NitroBoost `json:"premium_tier"`
	NitroBoosters uint64     `json:"premium_subscription_count,omitempty"`

	// Defaults to en-US, only set if guild has DISCOVERABLE
	PreferredLocale string `json:"preferred_locale"`
}

type Role struct {
	ID   Snowflake `json:"id,string"`
	Name string    `json:"name"`

	Color    Color `json:"color"`
	Hoist    bool  `json:"hoist"` // if the role is separated
	Position int   `json:"position"`

	Permissions Permissions `json:"permissions"`

	Managed     bool `json:"managed"`
	Mentionable bool `json:"mentionable"`
}

type Presence struct {
	User    User        `json:"user"`
	RoleIDs []Snowflake `json:"roles"`
}

type Member struct {
	User    User        `json:"user"`
	Nick    string      `json:"nick,omitempty"`
	RoleIDs []Snowflake `json:"roles"`

	Joined       Timestamp `json:"joined_at"`
	BoostedSince Timestamp `json:"premium_since,omitempty"`

	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`
}

type Ban struct {
	Reason string `json:"reason,omitempty"`
	User   User   `json:"user"`
}

type Integration struct {
	ID   Snowflake `json:"id"`
	Name string    `json:"name"`
	Type Service   `json:"type"`

	Enabled bool `json:"enabled"`
	Syncing bool `json:"syncing"`

	// used for subscribers
	RoleID Snowflake `json:"role_id"`

	ExpireBehavior    int `json:"expire_behavior"`
	ExpireGracePeriod int `json:"expire_grace_period"`

	User    User `json:"user"`
	Account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"account"`

	SyncedAt Timestamp `json:"synced_at"`
}

type GuildEmbed struct {
	Enabled   bool      `json:"enabled"`
	ChannelID Snowflake `json:"channel_id,omitempty"`
}
