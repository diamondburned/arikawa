package discord

// Invite represents a code that when used, adds a user to a guild or group
// DM channel.
//
// https://discord.com/developers/docs/resources/invite#invite-object
type Invite struct {
	// InviteMetadata contains extra information about the invite.
	// So far, this field is only available when fetching Channel- or
	// GuildInvites. Additionally the Uses field is filled when getting the
	// VanityURL of a guild.
	InviteMetadata
	// Guild is the partial guild this invite is for.
	Guild *Guild `json:"guild,omitempty"`
	// Inviter is the user who created the invite
	Inviter *User `json:"inviter,omitempty"`
	// Target is the target user for this invite.
	Target *User `json:"target_user,omitempty"`
	// Code is the invite code (unique ID).
	Code string `json:"code"`
	// Channel is the partial channel this invite is for.
	Channel Channel `json:"channel"`
	// ApproximatePresences is the approximate count of online members (only
	// present when Target is set).
	ApproximatePresences uint `json:"approximate_presence_count,omitempty"`
	// ApproximateMembers is the approximate count of total members
	ApproximateMembers uint `json:"approximate_member_count,omitempty"`
	// Target type is the type of user target for this invite.
	TargetType InviteUserType `json:"target_user_type,omitempty"`
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
	// CreatedAt is the time when this invite was created.
	CreatedAt Timestamp `json:"created_at"`
	// Uses is the number of times this invite has been used.
	Uses int `json:"uses"`
	// MaxUses is the maximum number of times this invite can be used.
	MaxUses int `json:"max_uses"`
	// MaxAge is the duration (in seconds) after which the invite expires.
	MaxAge Seconds `json:"max_age"`
	// Temporary specifies whether this invite only grants temporary membership
	Temporary bool `json:"temporary"`
}
