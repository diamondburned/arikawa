package discord

type User struct {
	ID            Snowflake `json:"id,string"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	Avatar        Hash      `json:"avatar"`

	// These fields may be omitted

	Bot bool `json:"bot,omitempty"`
	MFA bool `json:"mfa_enabled,omitempty"`

	DiscordSystem bool `json:"system,omitempty"`
	EmailVerified bool `json:"verified,omitempty"`

	Locale string `json:"locale,omitempty"`
	Email  string `json:"email,omitempty"`

	Flags UserFlags `json:"flags,omitempty"`
	Nitro UserNitro `json:"premium_type,omitempty"`
}

type UserFlags uint16

const (
	NoFlag UserFlags = 0

	DiscordEmployee UserFlags = 1 << iota
	DiscordPartner
	HypeSquadEvents
	BugHunter
	HouseBravery
	HouseBrilliance
	HouseBalance
	EarlySupporter
	TeamUser
	System
)

type UserNitro uint8

const (
	NoUserNitro UserNitro = iota
	NitroClassic
	NitroFull
)

type Connection struct {
	ID   Snowflake `json:"id"`
	Name string    `json:"name"`
	Type Service   `json:"type"`

	Revoked      bool `json:"revoked"`
	Verified     bool `json:"verified"`
	FriendSync   bool `json:"friend_sync"`
	ShowActivity bool `json:"show_activity"`

	Visibility ConnectionVisibility `json:"visibility"`

	// Only partial
	Integratioons []Integration `json:"integrations"`
}

type ConnectionVisibility uint8

const (
	ConnectionNotVisible ConnectionVisibility = iota
	ConnectionVisibleEveryone
)
