package discord

import (
	"strconv"
	"time"
)

type User struct {
	// may be ommited
	Email         string `json:"email,omitempty"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        Hash   `json:"avatar"`
	// may be ommited
	Locale string `json:"locale,omitempty"`
	ID     UserID `json:"id"`
	// may be ommited
	Flags UserFlags `json:"flags,omitempty"`
	// may be ommited
	PublicFlags UserFlags `json:"public_flags,omitempty"`
	// may be ommited
	MFA bool `json:"mfa_enabled,omitempty"`
	// may be ommited
	DiscordSystem bool `json:"system,omitempty"`
	// may be ommited
	EmailVerified bool `json:"verified,omitempty"`
	// may be ommited
	Bot bool `json:"bot,omitempty"`
	// may be ommited
	Nitro UserNitro `json:"premium_type,omitempty"`
}

// CreatedAt returns a time object representing when the user was created.
func (u User) CreatedAt() time.Time {
	return u.ID.Time()
}

// Mention returns a mention of the user.
func (u User) Mention() string {
	return u.ID.Mention()
}

// Tag returns a tag of the user.
func (u User) Tag() string {
	return u.Username + "#" + u.Discriminator
}

// AvatarURL returns the URL of the Avatar Image. It automatically detects a
// suitable type.
func (u User) AvatarURL() string {
	return u.AvatarURLWithType(AutoImage)
}

// AvatarURLWithType returns the URL of the Avatar Image using the passed type.
// If the user has no Avatar, his default avatar will be returned. This
// requires ImageType Auto or PNG
//
// Supported Image Types: PNG, JPEG, WebP, GIF (read above for caveat)
func (u User) AvatarURLWithType(t ImageType) string {
	if u.Avatar == "" {
		if t != PNGImage && t != AutoImage {
			return ""
		}

		disc, err := strconv.Atoi(u.Discriminator)
		if err != nil { // this should never happen
			return ""
		}
		picNo := strconv.Itoa(disc % 5)

		return "https://cdn.discordapp.com/embed/avatars/" + picNo + ".png"
	}

	return "https://cdn.discordapp.com/avatars/" + u.ID.String() + "/" + t.format(u.Avatar)
}

type UserFlags uint32

const NoFlag UserFlags = 0

const (
	Employee UserFlags = 1 << iota
	Partner
	HypeSquadEvents
	BugHunterLvl1
	_
	_
	HouseBravery
	HouseBrilliance
	HouseBalance
	EarlySupporter
	TeamUser
	_
	System
	_
	BugHunterLvl2
	_
	VerifiedBot
	VerifiedBotDeveloper
	CertifiedModerator
)

type UserNitro uint8

const (
	NoUserNitro UserNitro = iota
	NitroClassic
	NitroFull
)

type Connection struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Type         Service              `json:"type"`
	Integrations []Integration        `json:"integrations"` // Only partial
	Revoked      bool                 `json:"revoked"`
	Verified     bool                 `json:"verified"`
	FriendSync   bool                 `json:"friend_sync"`
	ShowActivity bool                 `json:"show_activity"`
	Visibility   ConnectionVisibility `json:"visibility"`
}

type ConnectionVisibility uint8

const (
	ConnectionNotVisible ConnectionVisibility = iota
	ConnectionVisibleEveryone
)

type Activity struct {
	// User only
	Timestamps *ActivityTimestamp `json:"timestamps,omitempty"`
	// User only
	Emoji *Emoji `json:"emoji,omitempty"`
	// User only
	Secrets *ActivitySecrets `json:"secrets,omitempty"`
	// User only
	Assets *ActivityAssets `json:"assets,omitempty"`
	// User only
	Party *ActivityParty `json:"party,omitempty"`
	// Undocumented fields
	// User only
	SyncID string `json:"sync_id,omitempty"`
	// User only
	State string `json:"state,omitempty"`
	URL   URL    `json:"url,omitempty"`
	// User only
	Details string `json:"details,omitempty"`
	Name    string `json:"name"`
	// Undocumented fields
	// User only
	SessionID string `json:"session_id,omitempty"`
	// User only
	AppID AppID `json:"application_id,omitempty"`
	// User only
	CreatedAt UnixTimestamp `json:"created_at,omitempty"`
	// User only
	Flags ActivityFlags `json:"flags,omitempty"`
	// User only
	Instance bool         `json:"instance,omitempty"`
	Type     ActivityType `json:"type"` // Undocumented fields
}

type ActivityType uint8

const (
	// Playing $name
	GameActivity ActivityType = iota
	// Streaming $details
	StreamingActivity
	// Listening to $name
	ListeningActivity
	// Watching $name
	WatchingActivity
	// $emoji $state
	CustomActivity
)

type ActivityFlags uint32

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

// A Relationship between the logged in user and the user in the struct. This
// struct is undocumented.
type Relationship struct {
	User   User             `json:"user"`
	UserID UserID           `json:"id"`
	Type   RelationshipType `json:"type"`
}

// RelationshipType is an enum for a relationship.
type RelationshipType uint8

const (
	_ RelationshipType = iota
	FriendRelationship
	BlockedRelationship
	IncomingFriendRequest
	SentFriendRequest
)
