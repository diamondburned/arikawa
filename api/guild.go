package api

import (
	"io"
	"net/url"

	"github.com/diamondburned/arikawa/discord" // for clarity
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json/option"
)

var EndpointGuilds = Endpoint + "guilds/"

// https://discordapp.com/developers/docs/resources/guild#create-guild-json-params
type CreateGuildData struct {
	// Name is the 	name of the guild (2-100 characters)
	Name string `json:"name"`
	// VoiceRegion is the voice region id.
	VoiceRegion string `json:"region,omitempty"`
	// Icon is the base64 128x128 image for the guild icon.
	Icon *Image `json:"image,omitempty"`

	// Verification is the 	verification level.
	Verification *discord.Verification `json:"verification_level,omitempty"`
	// Notification is the 	default message notification level.
	Notification *discord.Notification `json:"default_message_notifications,omitempty"`
	// ExplicitFilter is the explicit content filter level.
	ExplicitFilter *discord.ExplicitFilter `json:"explicit_content_filter,omitempty"`

	// Roles are the new guild roles.
	//
	// When using the roles parameter, the first member of the array is used to
	// change properties of the guild's @everyone role. If you are trying to
	// bootstrap a guild with additional roles, keep this in mind.
	//
	// When using the roles parameter, the required id field within each role
	// object is an integer placeholder, and will be replaced by the API upon
	// consumption. Its purpose is to allow you to overwrite a role's
	// permissions in a channel when also passing in channels with the channels
	// array.
	Roles []discord.Role `json:"roles,omitempty"`
	// Channels are the new guild's channels.
	// Assigning a channel to a channel category is not supported by this
	// endpoint, i.e. a channel can't have the parent_id field.
	//
	// When using the channels parameter, the position field is ignored,
	// and none of the default channels are created.
	//
	// When using the channels parameter, the id field within each channel
	// object may be set to an integer placeholder, and will be replaced by the
	// API upon consumption. Its purpose is to allow you to create
	// GUILD_CATEGORY channels by setting the parent_id field on any children
	// to the category's id field. Category channels must be listed before any
	// children.
	Channels []discord.Channel `json:"channels,omitempty"`

	// AFKChannelID is the id for the afk channel.
	AFKChannelID discord.Snowflake `json:"afk_channel_id,omitempty"`
	// AFKTimeout is the afk timeout in seconds.
	AFKTimeout option.Seconds `json:"afk_timeout,omitempty"`

	// SystemChannelID is the id of the channel where guild notices such as
	// welcome messages and boost events are posted.
	SystemChannelID discord.Snowflake `json:"system_channel_id,omitempty"`
}

// CreateGuild creates a new guild. Returns a guild object on success.
// Fires a Guild Create Gateway event.
//
// This endpoint can be used only by bots in less than 10 guilds.
func (c *Client) CreateGuild(data CreateGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "POST", Endpoint+"guilds", httputil.WithJSONBody(data))
}

// Guild returns the guild object for the given id.
// ApproximateMembers and ApproximatePresences will not be set.
func (c *Client) Guild(id discord.Snowflake) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "GET", EndpointGuilds+id.String())
}

// GuildWithCount returns the guild object for the given id.
// This will also set the ApproximateMembers and ApproximatePresences fields
// of the guild struct.
func (c *Client) GuildWithCount(id discord.Snowflake) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(
		&g, "GET",
		EndpointGuilds+id.String(),
		httputil.WithSchema(c, url.Values{
			"with_counts": {"true"},
		}),
	)
}

// Guilds returns all guilds, automatically paginating. Be careful, as this
// method may abuse the API by requesting thousands or millions of guilds. For
// lower-level access, use GuildsRange. Guilds returned have some fields
// filled only (ID, Name, Icon, Owner, Permissions).
//
// Max can be 0, in which case the function will try and fetch all guilds.
func (c *Client) Guilds(max uint) ([]discord.Guild, error) {
	var guilds []discord.Guild
	var after discord.Snowflake = 0

	const hardLimit int = 100

	unlimited := max == 0

	for fetch := uint(hardLimit); max > 0 || unlimited; fetch = uint(hardLimit) {
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

// GuildsBefore fetches guilds before the specified ID. Check GuildsRange.
func (c *Client) GuildsBefore(before discord.Snowflake, limit uint) ([]discord.Guild, error) {
	return c.GuildsRange(before, 0, limit)
}

// GuildsAfter fetches guilds after the specified ID. Check GuildsRange.
func (c *Client) GuildsAfter(after discord.Snowflake, limit uint) ([]discord.Guild, error) {
	return c.GuildsRange(0, after, limit)
}

// GuildsRange returns a list of partial guild objects the current user is a
// member of. Requires the guilds OAuth2 scope.
//
// This endpoint returns 100 guilds by default, which is the maximum number
// of guilds a non-bot user can join. Therefore, pagination is not needed
// for integrations that need to get a list of the users' guilds.
func (c *Client) GuildsRange(before, after discord.Snowflake, limit uint) ([]discord.Guild, error) {
	switch {
	case limit == 0:
		limit = 100
	case limit > 100:
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before,omitempty"`
		After  discord.Snowflake `schema:"after,omitempty"`

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

// LeaveGuild leaves a guild.
func (c *Client) LeaveGuild(id discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointMe+"/guilds/"+id.String())
}

// https://discordapp.com/developers/docs/resources/guild#modify-guild-json-params
type ModifyGuildData struct {
	// Name is the guild's name.
	Name string `json:"name,omitempty"`
	// Region is the guild's voice region id.
	Region option.NullableString `json:"region,omitempty"`

	// Verification is the verification level.
	//
	// This field is nullable.
	Verification *discord.Verification `json:"verification_level,omitempty"`
	// Notification is the default message notification level.
	//
	// This field is nullable.
	Notification *discord.Notification `json:"default_message_notifications,omitempty"`
	// ExplicitFilter is the explicit content filter level.
	//
	// This field is nullable.
	ExplicitFilter *discord.ExplicitFilter `json:"explicit_content_filter,omitempty"`

	// AFKChannelID is the id for the afk channel.
	//
	// This field is nullable.
	AFKChannelID discord.Snowflake `json:"afk_channel_id,string,omitempty"`
	// AFKTimeout is the afk timeout in seconds.
	AFKTimeout option.Seconds `json:"afk_timeout,omitempty"`
	// Icon is the base64 1024x1024 png/jpeg/gif image for the guild icon
	// (can be animated gif when the server has the ANIMATED_ICON feature).
	Icon *Image `json:"icon,omitempty"`
	// Splash is the base64 16:9 png/jpeg image for the guild splash
	// (when the server has the INVITE_SPLASH feature).
	Splash *Image `json:"splash,omitempty"`
	// Banner is the base64 16:9 png/jpeg image for the guild banner (when the
	// server has BANNER feature).
	Banner *Image `json:"banner,omitempty"`

	// OwnerID is the user id to transfer guild ownership to (must be owner).
	OwnerID discord.Snowflake `json:"owner_id,omitempty"`

	// SystemChannelID is the id of the channel where guild notices such as
	// welcome messages and boost events are posted.
	//
	// This field is nullable.
	SystemChannelID discord.Snowflake `json:"system_channel_id,omitempty"`
	// RulesChannelID is the id of the channel where "PUBLIC" guilds display
	// rules and/or guidelines.
	//
	// This field is nullable.
	RulesChannelID discord.Snowflake `json:"rules_channel_id,omitempty"`
	// PublicUpdatesChannelID is the id of the channel where admins and
	// moderators of "PUBLIC" guilds receive notices from Discord.
	//
	// This field is nullable.
	PublicUpdatesChannelID discord.Snowflake `json:"public_updates_channel_id,omitempty"`

	// PreferredLocale is the preferred locale of a "PUBLIC" guild used in
	// server discovery and notices from Discord.
	//
	// This defaults to "en-US".
	PreferredLocale option.NullableString `json:"preferred_locale,omitempty"`
}

// ModifyGuild modifies a guild's settings. Requires the MANAGE_GUILD permission.
// Fires a Guild Update Gateway event.
func (c *Client) ModifyGuild(id discord.Snowflake, data ModifyGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(
		&g, "PATCH",
		EndpointGuilds+id.String(),
		httputil.WithJSONBody(data),
	)

}

// DeleteGuild deletes a guild permanently. The User must be owner.
//
// Fires a Guild Delete Gateway event.
func (c *Client) DeleteGuild(id discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointGuilds+id.String())
}

// GuildVoiceRegions is the same as /voice, but returns VIP ones as well if
// available.
func (c *Client) VoiceRegionsGuild(guildID discord.Snowflake) ([]discord.VoiceRegion, error) {
	var vrs []discord.VoiceRegion
	return vrs, c.RequestJSON(&vrs, "GET", EndpointGuilds+guildID.String()+"/regions")
}

// https://discord.com/developers/docs/resources/audit-log#get-guild-audit-log-query-string-parameters
type AuditLogData struct {
	// UserID filters the log for actions made by a user.
	UserID discord.Snowflake `schema:"user_id,omitempty"`
	// ActionType is the type of audit log event.
	ActionType discord.AuditLogEvent `schema:"action_type,omitempty"`
	// Before filters the log before a certain entry ID.
	Before discord.Snowflake `schema:"before,omitempty"`
	// Limit limits how many entries are returned (default 50, minimum 1,
	// maximum 100).
	Limit uint `schema:"limit"`
}

// AuditLog returns an audit log object for the guild.
//
// Requires the VIEW_AUDIT_LOG permission.
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

// Integrations returns a list of integration objects for the guild.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) Integrations(guildID discord.Snowflake) ([]discord.Integration, error) {
	var ints []discord.Integration
	return ints, c.RequestJSON(&ints, "GET", EndpointGuilds+guildID.String()+"/integrations")
}

// AttachIntegration attaches an integration object from the current user to
// the guild.
//
// Requires the MANAGE_GUILD permission.
// Fires a Guild Integrations Update Gateway event.
func (c *Client) AttachIntegration(guildID,
	integrationID discord.Snowflake, integrationType discord.Service) error {

	var param struct {
		Type discord.Service   `json:"type"`
		ID   discord.Snowflake `json:"id"`
	}

	param.Type = integrationType
	param.ID = integrationID

	return c.FastRequest(
		"POST",
		EndpointGuilds+guildID.String()+"/integrations",
		httputil.WithJSONBody(param),
	)
}

// https://discord.com/developers/docs/resources/guild#modify-guild-integration-json-params
type ModifyIntegrationData struct {
	// ExpireBehavior is the behavior when an integration subscription lapses
	// (see the integration expire behaviors documentation).
	ExpireBehavior *discord.ExpireBehavior `json:"expire_behavior,omitempty"`
	// ExpireGracePeriod is the period (in days) where the integration will
	// ignore lapsed subscriptions.
	ExpireGracePeriod option.NullableInt `json:"expire_grace_period,omitempty"`
	// EnableEmoticons specifies whether emoticons should be synced for this
	// integration (twitch only currently).
	EnableEmoticons option.NullableBool `json:"enable_emoticons,omitempty"`
}

// ModifyIntegration modifies the behavior and settings of an integration
// object for the guild.
//
// Requires the MANAGE_GUILD permission.
// Fires a Guild Integrations Update Gateway event.
func (c *Client) ModifyIntegration(
	guildID, integrationID discord.Snowflake, data ModifyIntegrationData) error {
	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/integrations/"+integrationID.String(),
		httputil.WithJSONBody(data),
	)
}

// Sync an integration. Requires the MANAGE_GUILD permission.
func (c *Client) SyncIntegration(guildID, integrationID discord.Snowflake) error {
	return c.FastRequest(
		"POST",
		EndpointGuilds+guildID.String()+"/integrations/"+integrationID.String()+"/sync",
	)
}

// GuildEmbed returns the guild embed object.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) GuildEmbed(guildID discord.Snowflake) (*discord.GuildEmbed, error) {
	var ge *discord.GuildEmbed
	return ge, c.RequestJSON(&ge, "GET", EndpointGuilds+guildID.String()+"/embed")
}

// https://discord.com/developers/docs/resources/guild#guild-embed-object-guild-embed-structure
type ModifyGuildEmbedData struct {
	Enabled   option.Bool       `json:"enabled,omitempty"`
	ChannelID discord.Snowflake `json:"channel_id,omitempty"`
}

// ModifyGuildEmbed modifies the guild embed and updates the passed in
// GuildEmbed data.
//
// This method should be used with care: if you still want the embed enabled,
// you need to set the Enabled boolean, even if it's already enabled. If you
// don't, JSON will default it to false.
func (c *Client) ModifyGuildEmbed(guildID discord.Snowflake, data discord.GuildEmbed) error {
	return c.RequestJSON(&data, "PATCH", EndpointGuilds+guildID.String()+"/embed")
}

// GuildVanityURL returns *Invite for guilds that have that feature enabled,
// but only Code and Uses are filled. Code will be "" if a vanity url for the
// guild is not set.
//
// Requires MANAGE_GUILD.
func (c *Client) GuildVanityURL(guildID discord.Snowflake) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "GET", EndpointGuilds+guildID.String()+"/vanity-url")
}

// https://discord.com/developers/docs/resources/guild#get-guild-widget-image-widget-style-options
type GuildImageStyle string

const (
	// GuildShield is a shield style widget with Discord icon and guild members
	// online count.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=shield
	GuildShield GuildImageStyle = "shield"
	// GuildBanner1 is a large image with guild icon, name and online count.
	// "POWERED BY DISCORD" as the footer of the widget.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner1
	GuildBanner1 GuildImageStyle = "banner1"
	// GuildBanner2 is a smaller widget style with guild icon, name and online
	// count. Split on the right with Discord logo.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner2
	GuildBanner2 GuildImageStyle = "banner2"
	// GuildBanner3 is a large image with guild icon, name and online count. In
	// the footer, Discord logo on the left and "Chat Now" on the right.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner3
	GuildBanner3 GuildImageStyle = "banner3"
	// GuildBanner4 is a large Discord logo at the top of the widget.
	// Guild icon, name and online count in the middle portion of the widget
	// and a "JOIN MY SERVER" button at the bottom.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner4
	GuildBanner4 GuildImageStyle = "banner4"
)

// GuildImageURL returns a link to the PNG image widget for the guild.
//
// Requires no permissions or authentication.
func (c *Client) GuildImageURL(guildID discord.Snowflake, img GuildImageStyle) string {
	return EndpointGuilds + guildID.String() + "/widget.png?style=" + string(img)
}

// GuildImage returns a PNG image widget for the guild. Requires no permissions
// or authentication.
func (c *Client) GuildImage(guildID discord.Snowflake, img GuildImageStyle) (io.ReadCloser, error) {
	r, err := c.Request("GET", c.GuildImageURL(guildID, img))
	if err != nil {
		return nil, err
	}

	return r.GetBody(), nil
}
