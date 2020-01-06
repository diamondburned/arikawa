package api

import (
	"io"

	"github.com/diamondburned/arikawa/discord" // for clarity
	d "github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/httputil"
)

const EndpointGuilds = Endpoint + "guilds/"

// https://discordapp.com/developers/docs/resources/guild#create-guild-json-params
type CreateGuildData struct {
	Name string `json:"name"`
	Icon Image  `json:"image,omitempty"`

	// package dc is just package discord
	Verification   d.Verification   `json:"verification_level"`
	Notification   d.Notification   `json:"default_message_notifications"`
	ExplicitFilter d.ExplicitFilter `json:"explicit_content_filter"`

	// [0] (First entry) is ALWAYS @everyone.
	Roles []discord.Role `json:"roles,omitempty"`

	// Voice only
	VoiceRegion string `json:"region,omitempty"`

	// Partial, id field is ignored. Usually only Name and Type are changed.
	Channels []discord.Channel `json:"channels,omitempty"`
}

func (c *Client) CreateGuild(data CreateGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "POST", Endpoint+"guilds",
		httputil.WithJSONBody(c, data))
}

func (c *Client) Guild(guildID discord.Snowflake) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "GET", EndpointGuilds+guildID.String())
}

// https://discordapp.com/developers/docs/resources/guild#modify-guild-json-params
type ModifyGuildData struct {
	Name   string `json:"name,omitempty"`
	Region string `json:"region,omitempty"`
	Icon   Image  `json:"image,omitempty"`

	// package d is just package discord
	Verification   *d.Verification   `json:"verification_level,omitempty"`
	Notification   *d.Notification   `json:"default_message_notifications,omitempty"`
	ExplicitFilter *d.ExplicitFilter `json:"explicit_content_filter,omitempty"`

	AFKChannelID *d.Snowflake `json:"afk_channel_id,string,omitempty"`
	AFKTimeout   *d.Seconds   `json:"afk_timeout,omitempty"`

	OwnerID d.Snowflake `json:"owner_id,string,omitempty"`

	Splash Image `json:"splash,omitempty"`
	Banner Image `json:"banner,omitempty"`

	SystemChannelID d.Snowflake `json:"system_channel_id,string,omitempty"`
}

func (c *Client) ModifyGuild(
	guildID discord.Snowflake, data ModifyGuildData) (*discord.Guild, error) {

	var g *discord.Guild
	return g, c.RequestJSON(&g, "PATCH", EndpointGuilds+guildID.String(),
		httputil.WithJSONBody(c, data))
}

func (c *Client) DeleteGuild(guildID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointGuilds+guildID.String())
}

// Members returns maximum 1000 members.
func (c *Client) Members(guildID discord.Snowflake) ([]discord.Member, error) {
	var mems []discord.Member
	var after discord.Snowflake = 0

	for {
		m, err := c.MembersAfter(guildID, after, 1000)
		if err != nil {
			return mems, err
		}
		mems = append(mems, m...)

		if len(mems) < 1000 {
			break
		}

		after = mems[999].User.ID
	}

	return mems, nil
}

// MembersAfter returns a list of all guild members, from 1-1000 for limits. The
// default limit is 1 and the maximum limit is 1000.
func (c *Client) MembersAfter(guildID, after discord.Snowflake,
	limit uint) ([]discord.Member, error) {

	if limit == 0 {
		limit = 1
	}

	if limit > 1000 {
		limit = 1000
	}

	var param struct {
		After discord.Snowflake `schema:"after,omitempty"`

		Limit uint `schema:"limit"`
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

// AnyMemberData, all fields are optional.
type AnyMemberData struct {
	Nick string `json:"nick,omitempty"`
	Mute bool   `json:"mute,omitempty"`
	Deaf bool   `json:"deaf,omitempty"`

	Roles []discord.Snowflake `json:"roles,omitempty"`

	// Only for ModifyMember, requires MOVE_MEMBER
	VoiceChannel discord.Snowflake `json:"channel_id,omitempty"`
}

// AddMember requires access(Token).
func (c *Client) AddMember(guildID, userID discord.Snowflake,
	token string, data AnyMemberData) (*discord.Member, error) {

	// VoiceChannel doesn't belong here
	data.VoiceChannel = 0

	var param struct {
		Token string `json:"access_token"`
		AnyMemberData
	}

	param.Token = token
	param.AnyMemberData = data

	var mem *discord.Member
	return mem, c.RequestJSON(
		&mem, "PUT",
		EndpointGuilds+guildID.String()+"/members/"+userID.String(),
		httputil.WithJSONBody(c, param),
	)
}

func (c *Client) ModifyMember(
	guildID, userID discord.Snowflake, data AnyMemberData) error {

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/members/"+userID.String(),
		httputil.WithJSONBody(c, data),
	)
}

// ChangeOwnNickname only replies with the nickname back, so we're not even
// going to bother.
func (c *Client) ChangeOwnNickname(
	guildID discord.Snowflake, nick string) error {

	var param struct {
		Nick string `json:"nick"`
	}

	param.Nick = nick

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/members/@me/nick",
		httputil.WithJSONBody(c, param),
	)
}

func (c *Client) AddRole(guildID, userID, roleID discord.Snowflake) error {
	return c.FastRequest("PUT", EndpointGuilds+guildID.String()+
		"/members/"+userID.String()+
		"/roles/"+roleID.String())
}

func (c *Client) RemoveRole(guildID, userID, roleID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointGuilds+guildID.String()+
		"/members/"+userID.String()+
		"/roles/"+roleID.String())
}

// Kick requires KICK_MEMBERS.
func (c *Client) Kick(guildID, userID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/members/"+userID.String())
}

func (c *Client) Bans(guildID discord.Snowflake) ([]discord.Ban, error) {
	var bans []discord.Ban
	return bans, c.RequestJSON(&bans, "GET",
		EndpointGuilds+guildID.String()+"/bans")
}

func (c *Client) GetBan(
	guildID, userID discord.Snowflake) (*discord.Ban, error) {

	var ban *discord.Ban
	return ban, c.RequestJSON(&ban, "GET",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String())
}

// Ban requires the BAN_MEMBERS permission. Days is the days back for Discord
// to delete the user's message, maximum 7 days.
func (c *Client) Ban(
	guildID, userID discord.Snowflake, days uint, reason string) error {

	if days > 7 {
		days = 7
	}

	var param struct {
		DeleteDays uint   `schema:"delete_message_days,omitempty"`
		Reason     string `schema:"reason,omitempty"`
	}

	param.DeleteDays = days
	param.Reason = reason

	return c.FastRequest(
		"PUT",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String(),
		httputil.WithSchema(c, param),
	)
}

// Unban also requires BAN_MEMBERS.
func (c *Client) Unban(guildID, userID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String())
}

func (c *Client) Roles(guildID discord.Snowflake) ([]discord.Role, error) {
	var roles []discord.Role
	return roles, c.RequestJSON(&roles, "GET",
		EndpointGuilds+guildID.String()+"/roles")
}

type AnyRoleData struct {
	Name  string        `json:"name,omitempty"`  // "new role"
	Color discord.Color `json:"color,omitempty"` // 0
	Hoist bool          `json:"hoist,omitempty"` // false (show role separately)

	Mentionable bool                `json:"mentionable,omitempty"` // false
	Permissions discord.Permissions `json:"permissions,omitempty"` // @everyone
}

func (c *Client) CreateRole(
	guildID discord.Snowflake, data AnyRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "POST",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(c, data),
	)
}

func (c *Client) MoveRole(
	guildID, roleID discord.Snowflake, position int) ([]discord.Role, error) {

	var param struct {
		ID  discord.Snowflake `json:"id"`
		Pos int               `json:"position"`
	}

	param.ID = roleID
	param.Pos = position

	var roles []discord.Role
	return roles, c.RequestJSON(
		&roles, "PATCH",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(c, param),
	)
}

func (c *Client) ModifyRole(guildID, roleID discord.Snowflake,
	data AnyRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "PATCH",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
		httputil.WithJSONBody(c, data),
	)
}

func (c *Client) DeleteRole(guildID, roleID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String())
}

// PruneCount returns the number of members that would be removed in a prune
// operation. Requires KICK_MEMBERS. Days must be 1 or more, default 7.
func (c *Client) PruneCount(
	guildID discord.Snowflake, days uint) (uint, error) {

	if days == 0 {
		days = 7
	}

	var param struct {
		Days uint `schema:"days"`
	}

	param.Days = days

	var resp struct {
		Pruned uint `json:"pruned"`
	}

	return resp.Pruned, c.RequestJSON(
		&resp, "GET",
		EndpointGuilds+guildID.String()+"/prune",
		httputil.WithSchema(c, param),
	)
}

// Prune returns the number of members that is removed. Requires KICK_MEMBERS.
// Days must be 1 or more, default 7.
func (c *Client) Prune(
	guildID discord.Snowflake, days uint) (uint, error) {

	if days == 0 {
		days = 7
	}

	var param struct {
		Count    uint `schema:"count"`
		RetCount bool `schema:"compute_prune_count"`
	}

	param.Count = days
	param.RetCount = true // maybe expose this later?

	var resp struct {
		Pruned uint `json:"pruned"`
	}

	return resp.Pruned, c.RequestJSON(
		&resp, "POST",
		EndpointGuilds+guildID.String()+"/prune",
		httputil.WithSchema(c, param),
	)
}

// GuildVoiceRegions is the same as /voice, but returns VIP ones as well if
// available.
func (c *Client) VoiceRegionsGuild(
	guildID discord.Snowflake) ([]discord.VoiceRegion, error) {

	var vrs []discord.VoiceRegion
	return vrs, c.RequestJSON(&vrs, "GET",
		EndpointGuilds+guildID.String()+"/regions")
}

// Integrations requires MANAGE_GUILD.
func (c *Client) Integrations(
	guildID discord.Snowflake) ([]discord.Integration, error) {

	var ints []discord.Integration
	return ints, c.RequestJSON(&ints, "GET",
		EndpointGuilds+guildID.String()+"/integrations")
}

// AttachIntegration requires MANAGE_GUILD.
func (c *Client) AttachIntegration(guildID, integrationID discord.Snowflake,
	integrationType discord.IntegrationType) error {

	var param struct {
		Type discord.IntegrationType `json:"type"`
		ID   discord.Snowflake       `json:"id"`
	}

	return c.FastRequest(
		"POST",
		EndpointGuilds+guildID.String()+"/integrations",
		httputil.WithJSONBody(c, param),
	)
}

// ModifyIntegration requires MANAGE_GUILD.
func (c *Client) ModifyIntegration(guildID, integrationID discord.Snowflake,
	expireBehavior, expireGracePeriod int, emoticons bool) error {

	var param struct {
		ExpireBehavior    int  `json:"expire_behavior"`
		ExpireGracePeriod int  `json:"expire_grace_period"`
		EnableEmoticons   bool `json:"enable_emoticons"`
	}

	param.ExpireBehavior = expireBehavior
	param.ExpireGracePeriod = expireGracePeriod
	param.EnableEmoticons = emoticons

	return c.FastRequest("PATCH", EndpointGuilds+guildID.String()+
		"/integrations/"+integrationID.String(),
		httputil.WithSchema(c, param),
	)
}

func (c *Client) SyncIntegration(
	guildID, integrationID discord.Snowflake) error {

	return c.FastRequest("POST", EndpointGuilds+guildID.String()+
		"/integrations/"+integrationID.String()+"/sync")
}

func (c *Client) GuildEmbed(
	guildID discord.Snowflake) (*discord.GuildEmbed, error) {

	var ge *discord.GuildEmbed
	return ge, c.RequestJSON(&ge, "GET",
		EndpointGuilds+guildID.String()+"/embed")
}

// ModifyGuildEmbed should be used with care: if you still want the embed
// enabled, you need to set the Enabled boolean, even if it's already enabled.
// If you don't, JSON will default it to false.
func (c *Client) ModifyGuildEmbed(guildID discord.Snowflake,
	data discord.GuildEmbed) (*discord.GuildEmbed, error) {

	return &data, c.RequestJSON(&data, "PATCH",
		EndpointGuilds+guildID.String()+"/embed")
}

// GuildVanityURL returns *Invite, but only Code and Uses are filled. Requires
// MANAGE_GUILD.
func (c *Client) GuildVanityURL(
	guildID discord.Snowflake) (*discord.Invite, error) {

	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "GET",
		EndpointGuilds+guildID.String()+"/vanity-url")
}

type GuildImageType string

const (
	GuildShield  GuildImageType = "shield"
	GuildBanner1 GuildImageType = "banner1"
	GuildBanner2 GuildImageType = "banner2"
	GuildBanner3 GuildImageType = "banner3"
	GuildBanner4 GuildImageType = "banner4"
)

func (c *Client) GuildImageURL(
	guildID discord.Snowflake, img GuildImageType) string {

	return EndpointGuilds + guildID.String() +
		"/widget.png?style=" + string(img)
}

func (c *Client) GuildImage(
	guildID discord.Snowflake, img GuildImageType) (io.ReadCloser, error) {

	r, err := c.Request("GET", c.GuildImageURL(guildID, img))
	if err != nil {
		return nil, err
	}

	return r.Body, nil
}
