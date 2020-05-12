package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json/option"
)

// Adds a role to a guild member.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) AddRole(guildID, userID, roleID discord.Snowflake) error {
	return c.FastRequest(
		"PUT",
		EndpointGuilds+guildID.String()+"/members/"+userID.String()+"/roles/"+roleID.String(),
	)
}

// RemoveRole removes a role from a guild member.
//
// Requires the MANAGE_ROLES permission.
// Fires a Guild Member Update Gateway event.
func (c *Client) RemoveRole(guildID, userID, roleID discord.Snowflake) error {
	return c.FastRequest(
		"DELETE",
		EndpointGuilds+guildID.String()+"/members/"+userID.String()+"/roles/"+roleID.String(),
	)
}

// Roles returns a list of role objects for the guild.
func (c *Client) Roles(guildID discord.Snowflake) ([]discord.Role, error) {
	var roles []discord.Role
	return roles, c.RequestJSON(&roles, "GET", EndpointGuilds+guildID.String()+"/roles")
}

// https://discord.com/developers/docs/resources/guild#create-guild-role-json-params
type CreateRoleData struct {
	// Name is the 	name of the role.
	//
	// Default: "new role"
	Name string `json:"name,omitempty"`
	// Permissions is the bitwise value of the enabled/disabled permissions.
	//
	// Default: @everyone permissions in guild
	Permissions discord.Permissions `json:"permissions,omitempty"`
	// Color is the RGB color value of the role.
	//
	// Default: 0
	Color discord.Color `json:"color,omitempty"`
	// Hoist specifies whether the role should be displayed separately in the
	// sidebar.
	//
	// Default: false
	Hoist bool `json:"hoist,omitempty"`
	// Mentionable specifies whether the role should be mentionable.
	//
	// Default: false
	Mentionable bool `json:"mentionable,omitempty"`
}

// CreateRole creates a new role for the guild.
//
// Requires the MANAGE_ROLES permission.
// Fires a Guild Role Create Gateway event.
func (c *Client) CreateRole(
	guildID discord.Snowflake, data CreateRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "POST",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(data),
	)
}

// https://discord.com/developers/docs/resources/guild#modify-guild-role-positions-json-params
type MoveRoleData struct {
	// ID is the id of the role.
	ID discord.Snowflake `json:"id"`
	// Position is the sorting position of the role.
	Position option.NullableInt `json:"position,omitempty"`
}

// MoveRole modifies the positions of a set of role objects for the guild.
//
// Requires the MANAGE_ROLES permission.
// Fires multiple Guild Role Update Gateway events.
func (c *Client) MoveRole(guildID discord.Snowflake, data []MoveRoleData) ([]discord.Role, error) {
	var roles []discord.Role
	return roles, c.RequestJSON(
		&roles, "PATCH",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(data),
	)
}

// https://discord.com/developers/docs/resources/guild#modify-guild-role-json-params
type ModifyRoleData struct {
	// Name is the 	name of the role.
	Name option.NullableString `json:"name,omitempty"`
	// Permissions is the bitwise value of the enabled/disabled permissions.
	Permissions *discord.Permissions `json:"permissions,omitempty"`
	// Permissions is the bitwise value of the enabled/disabled permissions.
	Color option.NullableColor `json:"color,omitempty"`
	// Hoist specifies whether the role should be displayed separately in the
	// sidebar.
	Hoist option.NullableBool `json:"hoist,omitempty"`
	// Mentionable specifies whether the role should be mentionable.
	Mentionable option.NullableBool `json:"mentionable,omitempty"`
}

// ModifyRole modifies a guild role.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) ModifyRole(
	guildID, roleID discord.Snowflake,
	data ModifyRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "PATCH",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
		httputil.WithJSONBody(data),
	)
}

// DeleteRole deletes a guild role.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) DeleteRole(guildID, roleID discord.Snowflake) error {
	return c.FastRequest(
		"DELETE",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
	)
}
