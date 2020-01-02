package discord

type Invite struct {
	Code    string  `json:"code"`
	Channel Channel `json:"channel"`         // partial
	Guild   *Guild  `json:"guild,omitempty"` // partial

	ApproxMembers uint `json:"approximate_members_count,omitempty"`

	Target     *User          `json:"target_user,omitempty"` // partial
	TargetType InviteUserType `json:"target_user_type,omitempty"`

	// Only available if Target is
	ApproxPresences uint `json:"approximate_presence_count,omitempty"`
}

type InviteUserType uint8

const (
	InviteNormalUser InviteUserType = iota
	InviteUserStream
)
