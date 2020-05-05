package discord

import "strings"

// DefaultAvatarURL is the link to the default green avatar on Discord. It's
// returned from AvatarURL() if the user doesn't have an avatar.
var DefaultAvatarURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"

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
	if u.Avatar == "" {
		return DefaultAvatarURL
	}

	base := "https://cdn.discordapp.com"
	base += "/avatars/" + u.ID.String() + "/" + u.Avatar

	if strings.HasPrefix(u.Avatar, "a_") {
		return base + ".gif"
	} else {
		return base + ".png"
	}
}

type UserFlags uint32

const (
	NoFlag UserFlags = 0

	DiscordEmployee UserFlags = 1 << iota
	DiscordPartner
	HypeSquadEvents
	BugHunterLvl1
	HouseBravery
	HouseBrilliance
	HouseBalance
	EarlySupporter
	TeamUser
	System
	BugHunterLvl2
	VerifiedBot
	VerifiedBotDeveloper
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
	URL  URL          `json:"url,omitempty"`

	// User only

	CreatedAt  UnixTimestamp      `json:"created_at,omitempty"`
	Timestamps *ActivityTimestamp `json:"timestamps,omitempty"`

	ApplicationID Snowflake `json:"application_id,omitempty"`
	Details       string    `json:"details,omitempty"`
	State         string    `json:"state,omitempty"` // party status
	Emoji         *Emoji    `json:"emoji,omitempty"`

	Party   *ActivityParty   `json:"party,omitempty"`
	Assets  *ActivityAssets  `json:"assets,omitempty"`
	Secrets *ActivitySecrets `json:"secrets,omitempty"`

	Instance bool          `json:"instance,omitempty"`
	Flags    ActivityFlags `json:"flags,omitempty"`

	// Undocumented fields
	SyncID    string `json:"sync_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type ActivityType uint8

const (
	// Playing $name
	GameActivity ActivityType = iota
	// Streaming $details
	StreamingActivity
	// Listening to $name
	ListeningActivity
	_
	// $emoji $state
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

type ActivityTimestamp struct {
	Start UnixMsTimestamp `json:"start,omitempty"`
	End   UnixMsTimestamp `json:"end,omitempty"`
}

type ActivityParty struct {
	ID   string `json:"id,omitempty"`
	Size [2]int `json:"size,omitempty"` // [ current, max ]
}

type ActivityAssets struct {
	LargeImage string `json:"large_image,omitempty"` // id
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"` // id
	SmallText  string `json:"small_text,omitempty"`
}

type ActivitySecrets struct {
	Join     string `json:"join,omitempty"`
	Spectate string `json:"spectate,omitempty"`
	Match    string `json:"match,omitempty"`
}
