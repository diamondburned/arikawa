package gateway

import "github.com/diamondburned/arikawa/discord"

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
	URL  discord.URL  `json:"url"`

	// User only

	CreatedAt  discord.UnixTimestamp `json:"created_at"`
	Timestamps struct {
		Start discord.UnixMsTimestamp `json:"start,omitempty"`
		End   discord.UnixMsTimestamp `json:"end,omitempty"`
	} `json:"timestamps,omitempty"`

	ApplicationID discord.Snowflake `json:"application_id,omitempty"`
	Details       string            `json:"details,omitempty"`
	State         string            `json:"state,omitempty"` // party status
	Emoji         discord.Emoji     `json:"emoji,omitempty"`

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
