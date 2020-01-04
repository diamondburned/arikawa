package discord

// Guild.MaxPresences is 5000 when it's 0. Seriously. The fucker who thought of
// this API thought it was also a good idea to return "null" instead of
// "5000." What the fucK? It has the same byte count, thus the same size.
// Sending the number over means it could be changed anytime and not force
// the developers to recheck. You could make the change right today and it
// would just work on all the clients. But no. IT'S FUCKING NULL!
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

type ExplicitFilter uint8

const (
	NoContentFilter ExplicitFilter = iota
	MembersWithoutRoles
	AllMembers
)

type Notification uint8

const (
	AllMessages Notification = iota
	OnlyMentions
)

type Verification uint8

const (
	NoVerification Verification = iota
	// LowVerification requires a verified email
	LowVerification
	// MediumVerification requires the user be registered for at least 5
	// minutes.
	MediumVerification
	// HighVerification requires the member be in the server for more than 10
	// minutes.
	HighVerification
	// VeryHighVerification requires the member to have a verified phone
	// number.
	VeryHighVerification
)
