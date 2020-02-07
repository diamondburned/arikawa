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

// IconURL returns the URL to the guild icon. An empty string is removed if
// there's no icon.
func (g Guild) IconURL() string {
	if g.Icon == "" {
		return ""
	}

	base := "https://cdn.discordapp.com/icons/" +
		g.ID.String() + "/" + g.Icon

	if len(g.Icon) > 2 && g.Icon[:2] == "a_" {
		return base + ".gif"
	}

	return base + ".png"
}

// BannerURL returns the URL to the banner, which is the image on top of the
// channels list.
func (g Guild) BannerURL() string {
	if g.Banner == "" {
		return ""
	}

	return "https://cdn.discordapp.com/banners/" +
		g.ID.String() + "/" + g.Banner + ".png"
}

// SplashURL returns the URL to the guild splash, which is the invite page's
// background.
func (g Guild) SplashURL() string {
	if g.Splash == "" {
		return ""
	}

	return "https://cdn.discordapp.com/banners/" +
		g.ID.String() + "/" + g.Splash + ".png"
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

func (r Role) Mention() string {
	return "<&" + r.ID.String() + ">"
}

type Presence struct {
	User    User        `json:"user"`
	RoleIDs []Snowflake `json:"roles"`

	// These fields are only filled in gateway events, according to the
	// documentation.

	Nick    string    `json:"nick"`
	GuildID Snowflake `json:"guild_id"`

	PremiumSince Timestamp `json:"premium_since,omitempty"`

	Game       *Activity  `json:"game"`
	Activities []Activity `json:"activities"`

	Status       Status `json:"status"`
	ClientStatus struct {
		Desktop Status `json:"status,omitempty"`
		Mobile  Status `json:"mobile,omitempty"`
		Web     Status `json:"web,omitempty"`
	} `json:"client_status"`
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

func (m Member) Mention() string {
	return "<@!" + m.User.ID.String() + ">"
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

// DefaultMemberColor is the color used for members without colored roles.
var DefaultMemberColor Color = 0x0

func MemberColor(guild Guild, member Member) Color {
	var c = DefaultMemberColor
	var pos int

	for _, r := range guild.Roles {
		if r.Color > 0 && r.Position > pos {
			c = r.Color
			pos = r.Position
		}
	}

	return c
}
