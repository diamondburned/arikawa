package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var EndpointInvites = Endpoint + "invites/"

// Invite returns an invite object for the given code.
//
// ApproxMembers will not get filled.
func (c *Client) Invite(code string) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(
		&inv, "GET",
		EndpointInvites+code,
	)
}

// InviteWithCounts returns an invite object for the given code and fills
// ApproxMembers.
func (c *Client) InviteWithCounts(code string) (*discord.Invite, error) {
	var params struct {
		WithCounts bool `schema:"with_counts,omitempty"`
	}

	params.WithCounts = true

	var inv *discord.Invite
	return inv, c.RequestJSON(
		&inv, "GET",
		EndpointInvites+code,
		httputil.WithSchema(c, params),
	)
}

// ChannelInvites returns a list of invite objects (with invite metadata) for
// the channel. Only usable for guild channels.
//
// Requires the MANAGE_CHANNELS permission.
func (c *Client) ChannelInvites(channelID discord.ChannelID) ([]discord.Invite, error) {
	var invs []discord.Invite
	return invs, c.RequestJSON(&invs, "GET",
		EndpointChannels+channelID.String()+"/invites")
}

// GuildInvites returns a list of invite objects (with invite metadata) for the
// guild.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) GuildInvites(guildID discord.GuildID) ([]discord.Invite, error) {
	var invs []discord.Invite
	return invs, c.RequestJSON(&invs, "GET",
		EndpointGuilds+guildID.String()+"/invites")
}

// https://discord.com/developers/docs/resources/channel#create-channel-invite-json-params
type CreateInviteData struct {
	// MaxAge is the duration of invite in seconds before expiry, or 0 for
	// never.
	//
	// Default:	86400 (24 hours)
	MaxAge option.Uint `json:"max_age,omitempty"`
	// MaxUses is the max number of uses or 0 for unlimited.
	//
	// Default:	0
	MaxUses uint `json:"max_uses,omitempty"`
	// Temporary specifies whether this invite only grants temporary membership.
	//
	// Default:	false
	Temporary bool `json:"temporary,omitempty"`
	// Unique has the following behavior: if true, don't try to reuse a similar
	// invite (useful for creating many unique one time use invites).
	//
	// Default:	false
	Unique bool `json:"unique,omitempty"`

	AuditLogReason `json:"-"`
}

// CreateInvite creates a new invite object for the channel. Only usable for
// guild channels.
//
// Requires the CREATE_INSTANT_INVITE permission.
func (c *Client) CreateInvite(
	channelID discord.ChannelID, data CreateInviteData) (*discord.Invite, error) {

	var inv *discord.Invite
	return inv, c.RequestJSON(
		&inv, "POST",
		EndpointChannels+channelID.String()+"/invites",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// JoinedInvite is returned after joining an invite.
type JoinedInvite struct {
	Code      string          `json:"code"`
	NewMember bool            `json:"new_member"`
	Guild     discord.Guild   `json:"guild"`
	Channel   discord.Channel `json:"channel"` // id, name, type only
}

// JoinInvite joins a guild using the given invite code. This endpoint is
// undocumented.
func (c *Client) JoinInvite(code string) (*JoinedInvite, error) {
	var inv *JoinedInvite
	return inv, c.RequestJSON(&inv, "POST", EndpointInvites+code)
}

// DeleteInvite deletes an invite.
//
// Requires the MANAGE_CHANNELS permission on the channel this invite belongs
// to, or MANAGE_GUILD to remove any invite across the guild.
//
// Fires an Invite Delete Gateway event.
func (c *Client) DeleteInvite(code string, reason AuditLogReason) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(
		&inv,
		"DELETE", EndpointInvites+code,
		httputil.WithHeaders(reason.Header()),
	)
}
