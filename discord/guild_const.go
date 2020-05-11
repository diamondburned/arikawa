package discord

import (
	"github.com/diamondburned/arikawa/utils/json/enum"
)

// Guild.MaxPresences is 5000 when it's 0.
const DefaultMaxPresences = 5000

type NitroBoost uint8

const (
	NoNitroLevel NitroBoost = iota
	NitroLevel1
	NitroLevel2
	NitroLevel3
)

type MFALevel uint8

const (
	NoMFA MFALevel = iota
	ElevatedMFA
)

type GuildFeature string

const (
	// Guild has access to set an invite splash background
	InviteSplash GuildFeature = "INVITE_SPLASH"
	// Guild has access to set 384kbps bitrate in voice (previously VIP voice
	// servers)
	VIPRegions GuildFeature = "VIP_REGIONS"
	// Guild has access to set a vanity URL
	VanityURL GuildFeature = "VANITY_URL"
	// Guild is verified
	Verified GuildFeature = "VERIFIED"
	// Guild is partnered
	Partnered GuildFeature = "PARTNERED"
	// Guild is public
	Public GuildFeature = "PUBLIC"
	// Guild has access to use commerce features (i.e. create store channels)
	Commerce GuildFeature = "COMMERCE"
	// Guild has access to create news channels
	News GuildFeature = "NEWS"
	// Guild is able to be discovered in the directory
	Discoverable GuildFeature = "DISCOVERABLE"
	// Guild is able to be featured in the directory
	Featurable GuildFeature = "FEATURABLE"
	// Guild has access to set an animated guild icon
	AnimatedIcon GuildFeature = "ANIMATED_ICON"
	// Guild has access to set a guild banner image
	Banner GuildFeature = "BANNER"
)

// ExplicitFilter is the explicit content filter level of a guild.
type ExplicitFilter enum.Enum

var (
	// NullExplicitFilter serialized to JSON null.
	// This should only be used on nullable fields.
	NullExplicitFilter ExplicitFilter = enum.Null
	// NoContentFilter disables content filtering for the guild.
	NoContentFilter ExplicitFilter = 0
	// MembersWithoutRoles filters only members without roles.
	MembersWithoutRoles ExplicitFilter = 1
	// AllMembers enables content filtering for all members.
	AllMembers ExplicitFilter = 2
)

func (f *ExplicitFilter) UnmarshalJSON(b []byte) error {
	i, err := enum.FromJSON(b)
	*f = ExplicitFilter(i)

	return err
}

func (f ExplicitFilter) MarshalJSON() ([]byte, error) {
	return enum.ToJSON(enum.Enum(f)), nil
}

// Notification is the default message notification level of a guild.
type Notification enum.Enum

var (
	// NullNotification serialized to JSON null.
	// This should only be used on nullable fields.
	NullNotification Notification = enum.Null
	// AllMessages sends notifications for all messages.
	AllMessages Notification = 0
	// OnlyMentions sends notifications only on mention.
	OnlyMentions Notification = 1
)

func (n *Notification) UnmarshalJSON(b []byte) error {
	i, err := enum.FromJSON(b)
	*n = Notification(i)

	return err
}

func (n Notification) MarshalJSON() ([]byte, error) { return enum.ToJSON(enum.Enum(n)), nil }

// Verification is the verification level required for a guild.
type Verification enum.Enum

var (
	// NullVerification serialized to JSON null.
	// This should only be used on nullable fields.
	NullVerification Verification = enum.Null
	// NoVerification required no verification.
	NoVerification Verification = 0
	// LowVerification requires a verified email
	LowVerification Verification = 1
	// MediumVerification requires the user be registered for at least 5
	// minutes.
	MediumVerification Verification = 2
	// HighVerification requires the member be in the server for more than 10
	// minutes.
	HighVerification Verification = 3
	// VeryHighVerification requires the member to have a verified phone
	// number.
	VeryHighVerification Verification = 4
)

func (v *Verification) UnmarshalJSON(b []byte) error {
	i, err := enum.FromJSON(b)
	*v = Verification(i)

	return err
}

func (v Verification) MarshalJSON() ([]byte, error) { return enum.ToJSON(enum.Enum(v)), nil }

// Service is used for guild integrations and user connections.
type Service string

const (
	Twitch  Service = "twitch"
	YouTube Service = "youtube"
)

// ExpireBehavior is the integration expire behavior that regulates what happens, if a subscriber expires.
type ExpireBehavior uint8

var (
	// RemoveRole removes the role of the subscriber.
	RemoveRole ExpireBehavior = 0
	// Kick kicks the subscriber from the guild.
	Kick ExpireBehavior = 1
)
