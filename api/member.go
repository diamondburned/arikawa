package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/internal/intmath"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

const MaxMemberFetchLimit = 1000

// Member returns a guild member object for the specified user.
func (c *Client) Member(guildID discord.GuildID, userID discord.UserID) (*discord.Member, error) {
	var m *discord.Member
	return m, c.RequestJSON(&m, "GET", EndpointGuilds+guildID.String()+"/members/"+userID.String())
}

// Members returns a list of members of the guild with the passed id. This
// method automatically paginates until it reaches the passed limit, or, if the
// limit is set to 0, has fetched all members in the guild.
//
// As the underlying endpoint has a maximum of 1000 members per request, at
// maximum a total of limit/1000 rounded up requests will be made, although
// they may be less if no more members are available.
//
// When fetching the members, those with the smallest ID will be fetched first.
func (c *Client) Members(guildID discord.GuildID, limit uint) ([]discord.Member, error) {
	return c.MembersAfter(guildID, 0, limit)
}

// MembersAfter returns a list of members of the guild with the passed id. This
// method automatically paginates until it reaches the passed limit, or, if the
// limit is set to 0, has fetched all members with an id higher than after.
//
// As the underlying endpoint has a maximum of 1000 members per request, at
// maximum a total of limit/1000 rounded up requests will be made, although
// they may be less, if no more members are available.
func (c *Client) MembersAfter(
	guildID discord.GuildID, after discord.UserID, limit uint) ([]discord.Member, error) {

	mems := make([]discord.Member, 0, limit)

	fetch := uint(MaxMemberFetchLimit)

	unlimited := limit == 0

	for limit > 0 || unlimited {
		// Only fetch as much as we need. Since limit gradually decreases,
		// we only need to fetch intmath.Min(fetch, limit).
		if limit > 0 {
			fetch = uint(intmath.Min(MaxMemberFetchLimit, int(limit)))
			limit -= fetch
		}

		m, err := c.membersAfter(guildID, after, fetch)
		if err != nil {
			return mems, err
		}
		mems = append(mems, m...)

		// There aren't any to fetch, even if this is less than limit.
		if len(m) < MaxMemberFetchLimit {
			break
		}

		after = mems[len(mems)-1].User.ID
	}

	if len(mems) == 0 {
		return nil, nil
	}

	return mems, nil
}

func (c *Client) membersAfter(
	guildID discord.GuildID, after discord.UserID, limit uint) ([]discord.Member, error) {

	switch {
	case limit == 0:
		limit = 0
	case limit > 1000:
		limit = 1000
	}

	var param struct {
		After discord.UserID `schema:"after,omitempty"`
		Limit uint           `schema:"limit"`
	}

	param.Limit = limit
	param.After = after

	var mems []discord.Member
	return mems, c.RequestJSON(
		&mems, "GET",
		EndpointGuilds+guildID.String()+"/members",
		httputil.WithSchema(c, param),
	)
}

// https://discord.com/developers/docs/resources/guild#add-guild-member-json-params
type AddMemberData struct {
	// Token is an oauth2 access token granted with the guilds.join to the
	// bot's application for the user you want to add to the guild.
	Token string `json:"access_token"`
	// Nick is the value to set users nickname to.
	//
	// Requires MANAGE_NICKNAMES.
	Nick option.String `json:"nick,omitempty"`
	// Roles is an array of role ids the member is assigned.
	//
	// Requires MANAGE_ROLES.
	Roles *[]discord.RoleID `json:"roles,omitempty"`
	// Mute specifies whether the user is muted in voice channels.
	//
	// Requires MUTE_MEMBERS.
	Mute option.Bool `json:"mute,omitempty"`
	// Deaf specifies whether the user is deafened in voice channels.
	//
	// Requires DEAFEN_MEMBERS.
	Deaf option.Bool `json:"deaf,omitempty"`
}

// AddMember adds a user to the guild, provided you have a valid oauth2 access
// token for the user with the guilds.join scope. Returns a 201 Created with
// the guild member as the body, or 204 No Content if the user is already a
// member of the guild.
//
// Fires a Guild Member Add Gateway event.
//
// The Authorization header must be a Bot token (belonging to the same
// application used for authorization), and the bot must be a member of the
// guild with CREATE_INSTANT_INVITE permission.
func (c *Client) AddMember(
	guildID discord.GuildID, userID discord.UserID, data AddMemberData) (*discord.Member, error) {

	var mem *discord.Member
	return mem, c.RequestJSON(
		&mem, "PUT",
		EndpointGuilds+guildID.String()+"/members/"+userID.String(),
		httputil.WithJSONBody(data),
	)
}

// https://discord.com/developers/docs/resources/guild#add-guild-member-json-params
type ModifyMemberData struct {
	// Nick is the value to set users nickname to.
	//
	// Requires MANAGE_NICKNAMES.
	Nick option.String `json:"nick,omitempty"`
	// Roles is an array of role ids the member is assigned.
	//
	// Requires MANAGE_ROLES.
	Roles *[]discord.RoleID `json:"roles,omitempty"`
	// Mute specifies whether the user is muted in voice channels.
	//
	// Requires MUTE_MEMBERS.
	Mute option.Bool `json:"mute,omitempty"`
	// Deaf specifies whether the user is deafened in voice channels.
	//
	// Requires DEAFEN_MEMBERS.
	Deaf option.Bool `json:"deaf,omitempty"`

	// Voice channel is the id of channel to move user to (if they are
	// connected to voice).
	//
	// Requires MOVE_MEMBER
	VoiceChannel discord.ChannelID `json:"channel_id,omitempty"`

	AuditLogReason `json:"-"`
}

// ModifyMember modifies attributes of a guild member. If the channel_id is set
// to null, this will force the target user to be disconnected from voice.
//
// Fires a Guild Member Update Gateway event.
func (c *Client) ModifyMember(
	guildID discord.GuildID, userID discord.UserID, data ModifyMemberData) error {

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/members/"+userID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// https://discord.com/developers/docs/resources/guild#get-guild-prune-count-query-string-params
type PruneCountData struct {
	// Days is the number of days to count prune for (1 or more, default 7).
	Days uint `schema:"days"`
	// IncludedRoles are the role(s) to include.
	IncludedRoles []discord.RoleID `schema:"include_roles,omitempty"`
}

// PruneCount returns the number of members that would be removed in a prune
// operation. Days must be 1 or more, default 7.
//
// By default, prune will not remove users with roles. You can optionally
// include specific roles in your prune by providing the IncludedRoles
// parameter. Any inactive user that has a subset of the provided role(s)
// will be counted in the prune and users with additional roles will not.
//
// Requires KICK_MEMBERS.
func (c *Client) PruneCount(guildID discord.GuildID, data PruneCountData) (uint, error) {
	if data.Days == 0 {
		data.Days = 7
	}

	var resp struct {
		Pruned uint `json:"pruned"`
	}

	return resp.Pruned, c.RequestJSON(
		&resp, "GET",
		EndpointGuilds+guildID.String()+"/prune",
		httputil.WithSchema(c, data),
	)
}

// https://discord.com/developers/docs/resources/guild#begin-guild-prune-query-string-params
type PruneData struct {
	// Days is the number of days to prune (1 or more, default 7).
	Days uint `schema:"days"`
	// ReturnCount specifies whether 'pruned' is returned. Discouraged for
	// large guilds.
	ReturnCount bool `schema:"compute_prune_count"`
	// IncludedRoles are the role(s) to include.
	IncludedRoles []discord.RoleID `schema:"include_roles,omitempty"`

	AuditLogReason `schema:"-"`
}

// Prune begins a prune. Days must be 1 or more, default 7.
//
// By default, prune will not remove users with roles. You can optionally
// include specific roles in your prune by providing the IncludedRoles
// parameter. Any inactive user that has a subset of the provided role(s)
// will be included in the prune and users with additional roles will not.
//
// Requires KICK_MEMBERS.
//
// Fires multiple Guild Member Remove Gateway events.
func (c *Client) Prune(guildID discord.GuildID, data PruneData) (uint, error) {
	if data.Days == 0 {
		data.Days = 7
	}

	var resp struct {
		Pruned uint `json:"pruned"`
	}

	return resp.Pruned, c.RequestJSON(
		&resp, "POST",
		EndpointGuilds+guildID.String()+"/prune",
		httputil.WithSchema(c, data), httputil.WithHeaders(data.Header()),
	)
}

// Kick removes a member from a guild.
//
// Requires KICK_MEMBERS permission.
//
// Fires a Guild Member Remove Gateway event.
func (c *Client) Kick(
	guildID discord.GuildID, userID discord.UserID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE",
		EndpointGuilds+guildID.String()+"/members/"+userID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}

// Bans returns a list of ban objects for the users banned from this guild.
//
// Requires the BAN_MEMBERS permission.
func (c *Client) Bans(guildID discord.GuildID) ([]discord.Ban, error) {
	var bans []discord.Ban
	return bans, c.RequestJSON(
		&bans, "GET",
		EndpointGuilds+guildID.String()+"/bans",
	)
}

// GetBan returns a ban object for the given user.
//
// Requires the BAN_MEMBERS permission.
func (c *Client) GetBan(guildID discord.GuildID, userID discord.UserID) (*discord.Ban, error) {
	var ban *discord.Ban
	return ban, c.RequestJSON(
		&ban, "GET",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String(),
	)
}

// https://discord.com/developers/docs/resources/guild#create-guild-ban-query-string-params
type BanData struct {
	// DeleteDays is the number of days to delete messages for (0-7).
	DeleteDays option.Uint `schema:"delete_message_days,omitempty"`

	AuditLogReason `schema:"-"`
}

// Ban creates a guild ban, and optionally delete previous messages sent by the
// banned user.
//
// Requires the BAN_MEMBERS permission.
//
// Fires a Guild Ban Add Gateway event.
func (c *Client) Ban(guildID discord.GuildID, userID discord.UserID, data BanData) error {
	return c.FastRequest(
		"PUT",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String(),
		httputil.WithSchema(c, data), httputil.WithHeaders(data.Header()),
	)
}

// Unban removes the ban for a user.
//
// Requires the BAN_MEMBERS permissions.
//
// Fires a Guild Ban Remove Gateway event.
func (c *Client) Unban(
	guildID discord.GuildID, userID discord.UserID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointGuilds+guildID.String()+"/bans/"+userID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}
