package api

import (
	"github.com/diamondburned/arikawa/discord"
)

const EndpointInvites = Endpoint + "invites/"

// Still unsure what this is
type MetaInvite struct {
	Inviter discord.User `json:"user"`
	Uses    uint         `json:"uses"`
	MaxUses uint         `json:"max_uses"`

	MaxAge discord.Seconds `json:"max_age"`

	Temporary bool              `json:"temporary"`
	CreatedAt discord.Timestamp `json:"created_at"`
}

func (c *Client) Invite(code string) (*discord.Invite, error) {
	var params struct {
		WithCounts bool `json:"with_counts,omitempty"`
	}

	// Nothing says I can't!
	params.WithCounts = true

	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "GET", EndpointInvites+code)
}

// Invites is only for guild channels.
func (c *Client) Invites(channelID discord.Snowflake) ([]discord.Invite, error) {
	var invs []discord.Invite
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
	maxAge discord.Seconds, maxUses uint, temp, unique bool) (*discord.Invite, error) {

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

	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "POST",
		EndpointChannels+channelID.String()+"/invites")
}

// DeleteInvite requires either MANAGE_CHANNELS on the target channel, or
// MANAGE_GUILD to remove any invite in the guild.
func (c *Client) DeleteInvite(code string) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "DELETE", EndpointInvites+code)
}
