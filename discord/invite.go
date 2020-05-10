package discord

type Invite struct {
	Code    string  `json:"code"`
	Channel Channel `json:"channel"`         // partial
	Guild   *Guild  `json:"guild,omitempty"` // partial
	Inviter *User   `json:"inviter,omitempty"`

	ApproxMembers uint `json:"approximate_members_count,omitempty"`

	Target     *User          `json:"target_user,omitempty"` // partial
	TargetType InviteUserType `json:"target_user_type,omitempty"`

	// Only available if Target is
	ApproxPresences uint `json:"approximate_presence_count,omitempty"`

	InviteMetadata // only available when fetching ChannelInvites or GuildInvites
}

type InviteUserType uint8

const (
	InviteNormalUser InviteUserType = iota
	InviteUserStream
)

// Extra information about an invite, will extend the invite object.
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
