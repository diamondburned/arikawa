package discord

// Invite represents a code that when used, adds a user to a guild or group
// DM channel.
//
// https://discord.com/developers/docs/resources/invite#invite-object
type Invite struct {
	// Code is the invite code (unique ID).
	Code string `json:"code"`
	// Guild is the partial guild this invite is for.
	Guild *Guild `json:"guild,omitempty"`
	// Channel is the partial channel this invite is for.
	Channel Channel `json:"channel"`
	// Inviter is the user who created the invite
	Inviter *User `json:"inviter,omitempty"`

	// Target is the target user for this invite.
	Target *User `json:"target_user,omitempty"`
	// Target type is the type of user target for this invite.
	TargetType InviteUserType `json:"target_user_type,omitempty"`

	// ApproximatePresences is the approximate count of online members (only
	// present when Target is set).
	ApproximatePresences uint `json:"approximate_presence_count,omitempty"`
	// ApproximateMembers is the approximate count of total members
	ApproximateMembers uint `json:"approximate_member_count,omitempty"`

	// InviteMetadata contains extra information about the invite.
	// So far, this field is only available when fetching Channel- or
	// GuildInvites. Additionally the Uses field is filled when getting the
	// VanityURL of a guild.
	InviteMetadata
}

// https://discord.com/developers/docs/resources/invite#invite-object-target-user-types
type InviteUserType uint8

const (
	InviteNormalUser InviteUserType = iota
	InviteUserStream
)

// Extra information about an invite, will extend the invite object.
//
// https://discord.com/developers/docs/resources/invite#invite-metadata-object
type InviteMetadata struct {
	// Number of times this invite has been used
	Uses int `json:"uses"`
	// Max number of times this invite can be used
	MaxUses int `json:"max_uses"`
	// Duration (in seconds) after which the invite expires
	MaxAge Seconds `json:"max_age"`
	// Whether this invite only grants temporary membership
	Temporary bool `json:"temporary"`
	// When this invite was created
	CreatedAt Timestamp `json:"created_at"`
}
