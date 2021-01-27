package api

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
)

var EndpointApplications = Endpoint + "applications/"

type CreateCommandData struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Options     []discord.CommandOption `json:"options"`
}

func (c *Client) Commands(appID discord.AppID) ([]discord.Command, error) {
	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "GET",
		EndpointApplications+appID.String()+"/commands",
	)
}

func (c *Client) Command(
	appID discord.AppID, commandID discord.CommandID) ([]discord.Command, error) {

	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "GET",
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
	guildID discord.GuildID,
	commandID discord.CommandID) ([]discord.Command, error) {

	var cmds []discord.Command
	return cmds, c.RequestJSON(
		&cmds, "GET",
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
	appID discord.AppID,
	guildID discord.GuildID,
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
