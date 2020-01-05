package api

import (
	"github.com/diamondburned/arikawa/discord" // for clarity
	d "github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/httputil"
)

const EndpointGuilds = Endpoint + "guilds/"

// https://discordapp.com/developers/docs/resources/guild#create-guild-json-params
type CreateGuildData struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Icon   Image  `json:"image"`

	// package dc is just package discord
	Verification   d.Verification   `json:"verification_level"`
	Notification   d.Notification   `json:"default_message_notifications"`
	ExplicitFilter d.ExplicitFilter `json:"explicit_content_filter"`

	// [0] (First entry) is ALWAYS @everyone.
	Roles []discord.Role `json:"roles"`

	// Partial, id field is ignored. Usually only Name and Type are changed.
	Channels []discord.Channel `json:"channels"`
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
	Icon   *Image `json:"image,omitempty"`

	// package d is just package discord
	Verification   *d.Verification   `json:"verification_level,omitempty"`
	Notification   *d.Notification   `json:"default_message_notifications,omitempty"`
	ExplicitFilter *d.ExplicitFilter `json:"explicit_content_filter,omitempty"`

	AFKChannelID *d.Snowflake `json:"afk_channel_id,string,omitempty"`
	AFKTimeout   *d.Seconds   `json:"afk_timeout,omitempty"`

	OwnerID d.Snowflake `json:"owner_id,string,omitempty"`

	Splash *Image `json:"splash,omitempty"`
	Banner *Image `json:"banner,omitempty"`

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

func (c *Client) Members(guildID discord.Snowflake) ([]discord.Member, error) {
	var mems []discord.Member
	return mems, c.RequestJSON(&mems, "GET",
		EndpointGuilds+guildID.String()+"/members")
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

// Ban requires the BAN_MEMBERS permission. Days is the days back for Discord to
// delete the user's message, maximum 7 days.
func (c *Client) Ban(
	guildID, userID discord.Snowflake, days uint, reason string) error {

	if days > 7 {
		days = 7
	}

	var param struct {
		DeleteDays uint   `json:"delete_message_days,omitempty"`
		Reason     string `json:"reason,omitempty"`
	}

	param.DeleteDays = days
	param.Reason = reason

	return c.FastRequest(
		"PUT",
		EndpointGuilds+guildID.String()+"/bans/"+userID.String(),
		httputil.WithJSONBody(c, param),
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
