package api

import "git.sr.ht/~diamondburned/arikawa/discord"

const EndpointInvites = Endpoint + "invites/"

type MetaInvite struct {
	Inviter User `json:"user"`
	Uses    uint `json:"uses"`
	MaxUses uint `json:"max_uses"`

	MaxAge discord.Seconds `json:"max_age"`

	Temporary bool              `json:"temporary"`
	CreatedAt discord.Timestamp `json:"created_at"`
}

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

func (c *Client) Invite(code string) (*Invite, error) {
	var params struct {
		WithCounts bool `json:"with_counts,omitempty"`
	}

	// Nothing says I can't!
	params.WithCounts = true

	var inv *Invite
	return inv, c.RequestJSON(&inv, "GET", EndpointInvites+code)
}

// Invites is only for guild channels.
func (c *Client) Invites(channelID discord.Snowflake) ([]Invite, error) {
	var invs []Invite
	return invs, c.RequestJSON(&invs, "GET",
		EndpointChannels+channelID.String()+"/invites")
}

// CreateInvite is only for guild channels. This endpoint requires
// CREATE_INSTANT_INVITE.
//
// MaxAge is the duration before expiry, 0 for never. MaxUses is the maximum
// number of uses, 0 for unlimited. Temporary is whether this invite grants
// temporary membership. Unique, if true, tries not to reuse a similar invite,
// useful for creating unique one time use invites.
func (c *Client) CreateInvite(channelID discord.Snowflake,
	maxAge discord.Seconds, maxUses uint, temp, unique bool) (*Invite, error) {

	var params struct {
		MaxAge    uint `json:"max_age"`
		MaxUses   uint `json:"max_uses"`
		Temporary bool `json:"temporary"`
		Unique    bool `json:"unique"`
	}

	params.MaxAge = uint(maxAge)
	params.MaxUses = maxUses
	params.Temporary = temp
	params.Unique = unique

	var inv *Invite
	return inv, c.RequestJSON(&inv, "POST",
		EndpointChannels+channelID.String()+"/invites")
}

// DeleteInvite requires either MANAGE_CHANNELS on the target channel, or
// MANAGE_GUILD to remove any invite in the guild.
func (c *Client) DeleteInvite(code string) (*Invite, error) {
	var inv *Invite
	return inv, c.RequestJSON(&inv, "DELETE", EndpointInvites+code)
}
