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
