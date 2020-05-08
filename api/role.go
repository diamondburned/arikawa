package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

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

func (c *Client) CreateRole(guildID discord.Snowflake, data AnyRoleData) (*discord.Role, error) {
	var role *discord.Role
	return role, c.RequestJSON(
		&role, "POST",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(data),
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
		httputil.WithJSONBody(param),
	)
}

func (c *Client) ModifyRole(
	guildID, roleID discord.Snowflake,
	data AnyRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "PATCH",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
		httputil.WithJSONBody(data),
	)
}

func (c *Client) DeleteRole(guildID, roleID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String())
}
