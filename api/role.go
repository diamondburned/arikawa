package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

type AddRoleData struct {
	AuditLogReason
}

// AddRole adds a role to a guild member.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) AddRole(
	guildID discord.GuildID,
	userID discord.UserID, roleID discord.RoleID, data AddRoleData) error {

	return c.FastRequest(
		"PUT",
		EndpointGuilds+guildID.String()+"/members/"+userID.String()+"/roles/"+roleID.String(),
		httputil.WithHeaders(data.Header()),
	)
}

// RemoveRole removes a role from a guild member.
//
// Requires the MANAGE_ROLES permission.
//
// Fires a Guild Member Update Gateway event.
func (c *Client) RemoveRole(
	guildID discord.GuildID,
	userID discord.UserID, roleID discord.RoleID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE",
		EndpointGuilds+guildID.String()+"/members/"+userID.String()+"/roles/"+roleID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}

// Roles returns a list of role objects for the guild.
func (c *Client) Roles(guildID discord.GuildID) ([]discord.Role, error) {
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
	Permissions discord.Permissions `json:"permissions,string,omitempty"`
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

	AddRoleData `json:"-"`
}

// CreateRole creates a new role for the guild.
//
// Requires the MANAGE_ROLES permission.
//
// Fires a Guild Role Create Gateway event.
func (c *Client) CreateRole(guildID discord.GuildID, data CreateRoleData) (*discord.Role, error) {
	var role *discord.Role
	return role, c.RequestJSON(
		&role, "POST",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

type (
	MoveRolesData struct {
		Roles []MoveRoleData

		AuditLogReason
	}

	// https://discord.com/developers/docs/resources/guild#modify-guild-role-positions-json-params
	MoveRoleData struct {
		// ID is the id of the role.
		ID discord.RoleID `json:"id"`
		// Position is the sorting position of the role.
		Position option.NullableInt `json:"position,omitempty"`
	}
)

// MoveRoles modifies the positions of a set of role objects for the guild.
//
// Requires the MANAGE_ROLES permission.
//
// Fires multiple Guild Role Update Gateway events.
func (c *Client) MoveRoles(guildID discord.GuildID, data MoveRolesData) ([]discord.Role, error) {
	var roles []discord.Role
	return roles, c.RequestJSON(
		&roles, "PATCH",
		EndpointGuilds+guildID.String()+"/roles",
		httputil.WithJSONBody(data.Roles), httputil.WithHeaders(data.Header()),
	)
}

// https://discord.com/developers/docs/resources/guild#modify-guild-role-json-params
type ModifyRoleData struct {
	// Name is the 	name of the role.
	Name option.NullableString `json:"name,omitempty"`
	// Permissions is the bitwise value of the enabled/disabled permissions.
	Permissions *discord.Permissions `json:"permissions,string,omitempty"`
	// Permissions is the bitwise value of the enabled/disabled permissions.
	//
	// This value is nullable.
	Color discord.Color `json:"color,omitempty"`
	// Hoist specifies whether the role should be displayed separately in the
	// sidebar.
	Hoist option.NullableBool `json:"hoist,omitempty"`
	// Mentionable specifies whether the role should be mentionable.
	Mentionable option.NullableBool `json:"mentionable,omitempty"`

	AddRoleData `json:"-"`
}

// ModifyRole modifies a guild role.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) ModifyRole(
	guildID discord.GuildID, roleID discord.RoleID, data ModifyRoleData) (*discord.Role, error) {

	var role *discord.Role
	return role, c.RequestJSON(
		&role, "PATCH",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// DeleteRole deletes a guild role.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) DeleteRole(
	guildID discord.GuildID, roleID discord.RoleID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE",
		EndpointGuilds+guildID.String()+"/roles/"+roleID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}
