package api

import (
	"io"

	"github.com/diamondburned/arikawa/discord" // for clarity
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json"
)

var EndpointGuilds = Endpoint + "guilds/"

// https://discordapp.com/developers/docs/resources/guild#create-guild-json-params
type CreateGuildData struct {
	Name string `json:"name"`
	Icon Image  `json:"image,omitempty"`

	// package dc is just package discord
	Verification   discord.Verification   `json:"verification_level"`
	Notification   discord.Notification   `json:"default_message_notifications"`
	ExplicitFilter discord.ExplicitFilter `json:"explicit_content_filter"`

	// [0] (First entry) is ALWAYS @everyone.
	Roles []discord.Role `json:"roles,omitempty"`

	// Voice only
	VoiceRegion string `json:"region,omitempty"`

	// Partial, id field is ignored. Usually only Name and Type are changed.
	Channels []discord.Channel `json:"channels,omitempty"`
}

func (c *Client) CreateGuild(data CreateGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "POST", Endpoint+"guilds", httputil.WithJSONBody(data))
}

func (c *Client) Guild(id discord.Snowflake) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "GET", EndpointGuilds+id.String())
}

// Guilds returns all guilds, automatically paginating. Be careful, as this
// method may abuse the API by requesting thousands or millions of guilds. For
// lower-level access, usee GuildsRange. Guilds returned have some fields
// filled only (ID, Name, Icon, Owner, Permissions).
func (c *Client) Guilds(max uint) ([]discord.Guild, error) {
	var guilds []discord.Guild
	var after discord.Snowflake = 0

	const hardLimit int = 100

	for fetch := uint(hardLimit); max > 0; fetch = uint(hardLimit) {
		if max > 0 {
			if fetch > max {
				fetch = max
			}
			max -= fetch
		}

		g, err := c.GuildsAfter(after, fetch)
		if err != nil {
			return guilds, err
		}
		guilds = append(guilds, g...)

		if len(g) < hardLimit {
			break
		}

		after = g[hardLimit-1].ID
	}

	return guilds, nil
}

// GuildsBefore fetches guilds. Check GuildsRange.
func (c *Client) GuildsBefore(before discord.Snowflake, limit uint) ([]discord.Guild, error) {
	return c.GuildsRange(before, 0, limit)
}

// GuildsAfter fetches guilds. Check GuildsRange.
func (c *Client) GuildsAfter(after discord.Snowflake, limit uint) ([]discord.Guild, error) {
	return c.GuildsRange(0, after, limit)
}

// GuildsRange fetches guilds. The limit is 1-100.
func (c *Client) GuildsRange(before, after discord.Snowflake, limit uint) ([]discord.Guild, error) {
	if limit == 0 {
		limit = 100
	}

	if limit > 100 {
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before"`
		After  discord.Snowflake `schema:"after"`

		Limit uint `schema:"limit"`
	}

	param.Before = before
	param.After = after
	param.Limit = limit

	var gs []discord.Guild
	return gs, c.RequestJSON(
		&gs, "GET",
		EndpointMe+"/guilds",
		httputil.WithSchema(c, param),
	)
}

func (c *Client) LeaveGuild(id discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointMe+"/guilds/"+id.String())
}

// https://discordapp.com/developers/docs/resources/guild#modify-guild-json-params
type ModifyGuildData struct {
	Name   string            `json:"name,omitempty"`
	Region json.OptionString `json:"region,omitempty"`

	// package d is just package discord
	Verification   *discord.Verification   `json:"verification_level,omitempty"`
	Notification   *discord.Notification   `json:"default_message_notifications,omitempty"`
	ExplicitFilter *discord.ExplicitFilter `json:"explicit_content_filter,omitempty"`

	AFKChannelID discord.Snowflake `json:"afk_channel_id,string,omitempty"`
	AFKTimeout   discord.Seconds   `json:"afk_timeout,omitempty"`

	OwnerID discord.Snowflake `json:"owner_id,omitempty"`

	Icon   *Image `json:"icon,omitempty"`
	Splash *Image `json:"splash,omitempty"`
	Banner *Image `json:"banner,omitempty"`

	SystemChannelID        discord.Snowflake `json:"system_channel_id,omitempty"`
	RulesChannelID         discord.Snowflake `json:"rules_channel_id,omitempty"`
	PublicUpdatesChannelID discord.Snowflake `json:"public_updates_channel_id,omitempty"`

	PreferredLocale json.OptionString `json:"preferred_locale,omitempty"`
}

func (c *Client) ModifyGuild(id discord.Snowflake, data ModifyGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(
		&g, "PATCH",
		EndpointGuilds+id.String(),
		httputil.WithJSONBody(data),
	)
}

func (c *Client) DeleteGuild(id discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointGuilds+id.String())
}

// GuildVoiceRegions is the same as /voice, but returns VIP ones as well if
// available.
func (c *Client) VoiceRegionsGuild(guildID discord.Snowflake) ([]discord.VoiceRegion, error) {
	var vrs []discord.VoiceRegion
	return vrs, c.RequestJSON(&vrs, "GET", EndpointGuilds+guildID.String()+"/regions")
}

// AuditLogData contains query parameters used for AuditLog. All fields are
// optional.
type AuditLogData struct {
	// Filter the log for actions made by a user
	UserID discord.Snowflake `schema:"user_id,omitempty"`
	// The type of audit log event
	ActionType discord.AuditLogEvent `schema:"action_type,omitempty"`
	// Filter the log before a certain entry ID
	Before discord.Snowflake `schema:"before,omitempty"`
	// How many entries are returned (default 50, minimum 1, maximum 100)
	Limit uint `schema:"limit"`
}

// AuditLog returns an audit log object for the guild. Requires the
// VIEW_AUDIT_LOG permission.
func (c *Client) AuditLog(guildID discord.Snowflake, data AuditLogData) (*discord.AuditLog, error) {
	switch {
	case data.Limit == 0:
		data.Limit = 50
	case data.Limit > 100:
		data.Limit = 100
	}

	var audit *discord.AuditLog

	return audit, c.RequestJSON(
		&audit, "GET",
		EndpointGuilds+guildID.String()+"/audit-logs",
		httputil.WithSchema(c, data),
	)
}

// Integrations requires MANAGE_GUILD.
func (c *Client) Integrations(guildID discord.Snowflake) ([]discord.Integration, error) {
	var ints []discord.Integration
	return ints, c.RequestJSON(&ints, "GET", EndpointGuilds+guildID.String()+"/integrations")
}

// AttachIntegration requires MANAGE_GUILD.
func (c *Client) AttachIntegration(
	guildID, integrationID discord.Snowflake,
	integrationType discord.Service) error {

	var param struct {
		Type discord.Service   `json:"type"`
		ID   discord.Snowflake `json:"id"`
	}

	return c.FastRequest(
		"POST",
		EndpointGuilds+guildID.String()+"/integrations",
		httputil.WithJSONBody(param),
	)
}

// ModifyIntegration requires MANAGE_GUILD.
func (c *Client) ModifyIntegration(
	guildID, integrationID discord.Snowflake,
	expireBehavior, expireGracePeriod int, emoticons bool) error {

	var param struct {
		ExpireBehavior    int  `json:"expire_behavior"`
		ExpireGracePeriod int  `json:"expire_grace_period"`
		EnableEmoticons   bool `json:"enable_emoticons"`
	}

	param.ExpireBehavior = expireBehavior
	param.ExpireGracePeriod = expireGracePeriod
	param.EnableEmoticons = emoticons

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/integrations/"+integrationID.String(),
		httputil.WithSchema(c, param),
	)
}

func (c *Client) SyncIntegration(guildID, integrationID discord.Snowflake) error {
	return c.FastRequest("POST", EndpointGuilds+guildID.String()+
		"/integrations/"+integrationID.String()+"/sync")
}

func (c *Client) GuildEmbed(guildID discord.Snowflake) (*discord.GuildEmbed, error) {
	var ge *discord.GuildEmbed
	return ge, c.RequestJSON(&ge, "GET", EndpointGuilds+guildID.String()+"/embed")
}

// ModifyGuildEmbed modifies the guild embed and updates the passed in
// GuildEmbed data.
//
// This method should be used with care: if you still want the embed enabled,
// you need to set the Enabled boolean, even if it's already enabled. If you
// don't, JSON will default it to false.
func (c *Client) ModifyGuildEmbed(guildID discord.Snowflake, data *discord.GuildEmbed) error {
	return c.RequestJSON(&data, "PATCH", EndpointGuilds+guildID.String()+"/embed")
}

// GuildVanityURL returns *Invite, but only Code and Uses are filled. Requires
// MANAGE_GUILD.
func (c *Client) GuildVanityURL(guildID discord.Snowflake) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "GET", EndpointGuilds+guildID.String()+"/vanity-url")
}

type GuildImageType string

const (
	GuildShield  GuildImageType = "shield"
	GuildBanner1 GuildImageType = "banner1"
	GuildBanner2 GuildImageType = "banner2"
	GuildBanner3 GuildImageType = "banner3"
	GuildBanner4 GuildImageType = "banner4"
)

func (c *Client) GuildImageURL(guildID discord.Snowflake, img GuildImageType) string {
	return EndpointGuilds + guildID.String() + "/widget.png?style=" + string(img)
}

func (c *Client) GuildImage(guildID discord.Snowflake, img GuildImageType) (io.ReadCloser, error) {
	r, err := c.Request("GET", c.GuildImageURL(guildID, img))
	if err != nil {
		return nil, err
	}

	return r.GetBody(), nil
}
