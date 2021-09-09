package api

import (
	"encoding/json"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

var EndpointApplications = Endpoint + "applications/"

// https://discord.com/developers/docs/interactions/slash-commands#create-global-application-command-json-params
type CreateCommandData struct {
	Name                string                  `json:"name"`
	Description         string                  `json:"description"`
	Options             []discord.CommandOption `json:"options,omitempty"`
	NoDefaultPermission bool                    `json:"-"`
	Type                discord.CommandType     `json:"type,omitempty"`
}

func (c CreateCommandData) MarshalJSON() ([]byte, error) {
	type RawCreateCommandData CreateCommandData
	cmd := struct {
		RawCreateCommandData
		DefaultPermission bool `json:"default_permission"`
	}{RawCreateCommandData: (RawCreateCommandData)(c)}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	cmd.DefaultPermission = !c.NoDefaultPermission

	return json.Marshal(cmd)
}

func (c *CreateCommandData) UnmarshalJSON(data []byte) error {
	type RawCreateCommandData CreateCommandData
	cmd := struct {
		*RawCreateCommandData
		DefaultPermission bool `json:"default_permission"`
	}{RawCreateCommandData: (*RawCreateCommandData)(c)}
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	c.NoDefaultPermission = !cmd.DefaultPermission

	// Discord defaults type to 1 if omitted.
	if c.Type == 0 {
		c.Type = discord.ChatInputCommand
	}

	return nil
}

func (c *Client) Commands(appID discord.AppID) ([]discord.Command, error) {
	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "GET",
		EndpointApplications+appID.String()+"/commands",
	)
}

func (c *Client) Command(
	appID discord.AppID, commandID discord.CommandID) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "GET",
		EndpointApplications+appID.String()+"/commands/"+commandID.String(),
	)
}

func (c *Client) CreateCommand(
	appID discord.AppID, data CreateCommandData) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "POST",
		EndpointApplications+appID.String()+"/commands",
		httputil.WithJSONBody(data),
	)
}

func (c *Client) EditCommand(
	appID discord.AppID,
	commandID discord.CommandID, data CreateCommandData) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "PATCH",
		EndpointApplications+appID.String()+"/commands/"+commandID.String(),
		httputil.WithJSONBody(data),
	)
}

func (c *Client) DeleteCommand(appID discord.AppID, commandID discord.CommandID) error {
	return c.FastRequest(
		"DELETE",
		EndpointApplications+appID.String()+"/commands/"+commandID.String(),
	)
}

// BulkOverwriteCommands takes a slice of application commands, overwriting
// existing commands that are registered globally for this application. Updates
// will be available in all guilds after 1 hour.
//
// Commands that do not already exist will count toward daily application
// command create limits.
func (c *Client) BulkOverwriteCommands(
	appID discord.AppID, commands []discord.Command) ([]discord.Command, error) {

	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "PUT",
		EndpointApplications+appID.String()+"/commands",
		httputil.WithJSONBody(commands))
}

func (c *Client) GuildCommands(
	appID discord.AppID, guildID discord.GuildID) ([]discord.Command, error) {

	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "GET",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+"/commands",
	)
}

func (c *Client) GuildCommand(
	appID discord.AppID,
	guildID discord.GuildID, commandID discord.CommandID) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "GET",
		EndpointApplications+appID.String()+
			"/guilds/"+guildID.String()+
			"/commands/"+commandID.String(),
	)
}

func (c *Client) CreateGuildCommand(
	appID discord.AppID,
	guildID discord.GuildID, data CreateCommandData) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "POST",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+"/commands",
		httputil.WithJSONBody(data),
	)
}

func (c *Client) EditGuildCommand(
	appID discord.AppID, guildID discord.GuildID,
	commandID discord.CommandID, data CreateCommandData) (*discord.Command, error) {

	var cmd *discord.Command
	return cmd, c.RequestJSON(
		&cmd, "PATCH",
		EndpointApplications+appID.String()+
			"/guilds/"+guildID.String()+
			"/commands/"+commandID.String(),
		httputil.WithJSONBody(data),
	)
}

func (c *Client) DeleteGuildCommand(
	appID discord.AppID, guildID discord.GuildID, commandID discord.CommandID) error {

	return c.FastRequest(
		"DELETE",
		EndpointApplications+appID.String()+
			"/guilds/"+guildID.String()+
			"/commands/"+commandID.String(),
	)
}

// BulkOverwriteGuildCommands takes a slice of application commands,
// overwriting existing commands that are registered for the guild.
func (c *Client) BulkOverwriteGuildCommands(
	appID discord.AppID,
	guildID discord.GuildID, commands []discord.Command) ([]discord.Command, error) {

	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "PUT",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+"/commands",
		httputil.WithJSONBody(commands))
}

// GuildCommandPermissions fetches command permissions for all commands for the
// application in a guild.
func (c *Client) GuildCommandPermissions(
	appID discord.AppID, guildID discord.GuildID) ([]discord.GuildCommandPermissions, error) {

	var perms []discord.GuildCommandPermissions
	return perms, c.RequestJSON(
		&perms, "GET",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+"/commands/permissions",
	)
}

// CommandPermissions fetches command permissions for a specific command for
// the application in a guild.
func (c *Client) CommandPermissions(
	appID discord.AppID, guildID discord.GuildID,
	commandID discord.CommandID) (*discord.GuildCommandPermissions, error) {

	var perms *discord.GuildCommandPermissions
	return perms, c.RequestJSON(
		&perms, "GET",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+
			"/commands/"+commandID.String()+"/permissions",
	)
}

type editCommandPermissionsData struct {
	Permissions []discord.CommandPermissions `json:"permissions"`
}

// EditCommandPermissions edits command permissions for a specific command for
// the application in a guild. Up to 10 permission overwrites can be added for
// a command.
//
// Existing permissions for the command will be overwritten in that guild.
// Deleting or renaming a command will permanently delete all permissions for
// that command.
func (c *Client) EditCommandPermissions(
	appID discord.AppID, guildID discord.GuildID, commandID discord.CommandID,
	permissions []discord.CommandPermissions) (*discord.GuildCommandPermissions, error) {

	data := editCommandPermissionsData{Permissions: permissions}

	var perms *discord.GuildCommandPermissions
	return perms, c.RequestJSON(
		&perms, "PUT",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+
			"/commands/"+commandID.String()+"/permissions",
		httputil.WithJSONBody(data),
	)
}

// https://discord.com/developers/docs/interactions/slash-commands#application-command-permissions-object-guild-application-command-permissions-structure
type BatchEditCommandPermissionsData struct {
	ID          discord.CommandID            `json:"id"`
	Permissions []discord.CommandPermissions `json:"permissions"`
}

// BatchEditCommandPermissions batch edits permissions for all commands in a
// guild. Up to 10 permission overwrites can be added for a command.
//
// Existing permissions for the command will be overwritten in that guild.
// Deleting or renaming a command will permanently delete all permissions for
// that command.
func (c *Client) BatchEditCommandPermissions(
	appID discord.AppID, guildID discord.GuildID,
	data []BatchEditCommandPermissionsData) ([]discord.GuildCommandPermissions, error) {

	var perms []discord.GuildCommandPermissions
	return perms, c.RequestJSON(
		&perms, "PUT",
		EndpointApplications+appID.String()+"/guilds/"+guildID.String()+"/commands/permissions",
		httputil.WithJSONBody(data),
	)
}
