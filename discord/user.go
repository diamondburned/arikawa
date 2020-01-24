package discord

import "strings"

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

func (u User) Mention() string {
	return "<@" + u.ID.String() + ">"
}

func (u User) AvatarURL() string {
	base := "https://cdn.discordapp.com"

	if u.Avatar == "" {
		return base + "/embed/avatars/" + u.Discriminator + ".png"
	}

	base += "/avatars/" + u.ID.String() + "/" + u.Avatar

	if strings.HasPrefix(u.Avatar, "a_") {
		return base + ".gif"
	} else {
		return base + ".png"
	}
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
	Integrations []Integration `json:"integrations"`
}

type ConnectionVisibility uint8

const (
	ConnectionNotVisible ConnectionVisibility = iota
	ConnectionVisibleEveryone
)

type Status string

const (
	UnknownStatus      Status = ""
	OnlineStatus       Status = "online"
	DoNotDisturbStatus Status = "dnd"
	IdleStatus         Status = "idle"
	InvisibleStatus    Status = "invisible"
	OfflineStatus      Status = "offline"
)

type Activity struct {
	Name string       `json:"name"`
	Type ActivityType `json:"type"`
	URL  URL          `json:"url"`

	// User only

	CreatedAt  UnixTimestamp `json:"created_at"`
	Timestamps struct {
		Start UnixMsTimestamp `json:"start,omitempty"`
		End   UnixMsTimestamp `json:"end,omitempty"`
	} `json:"timestamps,omitempty"`

	ApplicationID Snowflake `json:"application_id,omitempty"`
	Details       string    `json:"details,omitempty"`
	State         string    `json:"state,omitempty"` // party status
	Emoji         Emoji     `json:"emoji,omitempty"`

	Party struct {
		ID   string `json:"id,omitempty"`
		Size [2]int `json:"size,omitempty"` // [ current, max ]
	} `json:"party,omitempty"`

	Assets struct {
		LargeImage string `json:"large_image,omitempty"` // id
		LargeText  string `json:"large_text,omitempty"`
		SmallImage string `json:"small_image,omitempty"` // id
		SmallText  string `json:"small_text,omitempty"`
	} `json:"assets,omitempty"`

	Secrets struct {
		Join     string `json:"join,omitempty"`
		Spectate string `json:"spectate,omitempty"`
		Match    string `json:"match,omitempty"`
	} `json:"secrets,omitempty"`

	Instance bool          `json:"instance,omitempty"`
	Flags    ActivityFlags `json:"flags,omitempty"`
}

type ActivityType uint8

const (
	// Playing $name
	GameActivity ActivityType = iota
	// Streaming $details
	StreamingActivity
	// Listening to $name
	ListeningActivity
	// $emoji $name
	CustomActivity
)

type ActivityFlags uint8

const (
	InstanceActivity ActivityFlags = 1 << iota
	JoinActivity
	SpectateActivity
	JoinRequestActivity
	SyncActivity
	PlayActivity
)
