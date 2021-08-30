package api

import (
	"io"
	"net/url"

	"github.com/diamondburned/arikawa/v3/discord" // for clarity
	"github.com/diamondburned/arikawa/v3/internal/intmath"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// MaxGuildFetchLimit is the limit of max guilds per request, as imposed by
// Discord.
const MaxGuildFetchLimit = 100

var EndpointGuilds = Endpoint + "guilds/"

// https://discord.com/developers/docs/resources/guild#create-guild-json-params
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
	AFKChannelID discord.ChannelID `json:"afk_channel_id,omitempty"`
	// AFKTimeout is the afk timeout in seconds.
	AFKTimeout discord.OptionalSeconds `json:"afk_timeout,omitempty"`

	// SystemChannelID is the id of the channel where guild notices such as
	// welcome messages and boost events are posted.
	SystemChannelID discord.ChannelID `json:"system_channel_id,omitempty"`
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
//
// ApproximateMembers and ApproximatePresences will not be set.
func (c *Client) Guild(id discord.GuildID) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(&g, "GET", EndpointGuilds+id.String())
}

// GuildPreview returns the guild preview object for the given id, even if the
// user is not in the guild.
//
// This endpoint is only for public guilds.
func (c *Client) GuildPreview(id discord.GuildID) (*discord.GuildPreview, error) {
	var g *discord.GuildPreview
	return g, c.RequestJSON(&g, "GET", EndpointGuilds+id.String()+"/preview")
}

// GuildWithCount returns the guild object for the given id. This will also
// set the ApproximateMembers and ApproximatePresences fields of the guild
// struct.
func (c *Client) GuildWithCount(id discord.GuildID) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(
		&g, "GET",
		EndpointGuilds+id.String(),
		httputil.WithSchema(c, url.Values{
			"with_counts": {"true"},
		}),
	)
}

// Guilds returns a list of partial guild objects the current user is a member
// of. This method automatically paginates until it reaches the passed limit,
// or, if the limit is set to 0, has fetched all guilds the user has joined.
//
// As the underlying endpoint has a maximum of 100 guilds per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
//
// When fetching the guilds, those with the smallest ID will be fetched first.
//
// Also note that 100 is the maximum number of guilds a non-bot user can join.
// Therefore, pagination is not needed for integrations that need to get a list
// of the users' guilds.
//
// Requires the guilds OAuth2 scope.
func (c *Client) Guilds(limit uint) ([]discord.Guild, error) {
	return c.GuildsAfter(0, limit)
}

// GuildsBefore returns a list of partial guild objects the current user is a
// member of. This method automatically paginates until it reaches the
// passed limit, or, if the limit is set to 0, has fetched all guilds with an
// id smaller than before.
//
// As the underlying endpoint has a maximum of 100 guilds per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
//
// Requires the guilds OAuth2 scope.
func (c *Client) GuildsBefore(before discord.GuildID, limit uint) ([]discord.Guild, error) {
	guilds := make([]discord.Guild, 0, limit)

	fetch := uint(MaxGuildFetchLimit)

	unlimited := limit == 0

	for limit > 0 || unlimited {
		if limit > 0 {
			// Only fetch as much as we need. Since limit gradually decreases,
			// we only need to fetch intmath.Min(fetch, limit).
			fetch = uint(intmath.Min(MaxGuildFetchLimit, int(limit)))
			limit -= fetch
		}

		g, err := c.guildsRange(before, 0, fetch)
		if err != nil {
			return guilds, err
		}
		guilds = append(g, guilds...)

		if len(g) < MaxGuildFetchLimit {
			break
		}

		before = g[0].ID
	}

	if len(guilds) == 0 {
		return nil, nil
	}

	return guilds, nil
}

// GuildsAfter returns a list of partial guild objects the current user is a
// member of. This method automatically paginates until it reaches the
// passed limit, or, if the limit is set to 0, has fetched all guilds with an
// id higher than after.
//
// As the underlying endpoint has a maximum of 100 guilds per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more guilds are available.
//
// Requires the guilds OAuth2 scope.
func (c *Client) GuildsAfter(after discord.GuildID, limit uint) ([]discord.Guild, error) {
	guilds := make([]discord.Guild, 0, limit)

	fetch := uint(MaxGuildFetchLimit)

	unlimited := limit == 0

	for limit > 0 || unlimited {
		if limit > 0 {
			// Only fetch as much as we need. Since limit gradually decreases,
			// we only need to fetch intmath.Min(fetch, limit).
			fetch = uint(intmath.Min(MaxGuildFetchLimit, int(limit)))
			limit -= fetch
		}

		g, err := c.guildsRange(0, after, fetch)
		if err != nil {
			return guilds, err
		}
		guilds = append(guilds, g...)

		if len(g) < MaxGuildFetchLimit {
			break
		}

		after = g[len(g)-1].ID
	}

	if len(guilds) == 0 {
		return nil, nil
	}

	return guilds, nil
}

func (c *Client) guildsRange(before, after discord.GuildID, limit uint) ([]discord.Guild, error) {
	var param struct {
		Before discord.GuildID `schema:"before,omitempty"`
		After  discord.GuildID `schema:"after,omitempty"`

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
func (c *Client) LeaveGuild(id discord.GuildID) error {
	return c.FastRequest("DELETE", EndpointMe+"/guilds/"+id.String())
}

// https://discord.com/developers/docs/resources/guild#modify-guild-json-params
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
	AFKChannelID discord.ChannelID `json:"afk_channel_id,string,omitempty"`
	// AFKTimeout is the afk timeout in seconds.
	AFKTimeout discord.OptionalSeconds `json:"afk_timeout,omitempty"`
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
	OwnerID discord.UserID `json:"owner_id,omitempty"`

	// SystemChannelID is the id of the channel where guild notices such as
	// welcome messages and boost events are posted.
	//
	// This field is nullable.
	SystemChannelID discord.ChannelID `json:"system_channel_id,omitempty"`
	// RulesChannelID is the id of the channel where "PUBLIC" guilds display
	// rules and/or guidelines.
	//
	// This field is nullable.
	RulesChannelID discord.ChannelID `json:"rules_channel_id,omitempty"`
	// PublicUpdatesChannelID is the id of the channel where admins and
	// moderators of "PUBLIC" guilds receive notices from Discord.
	//
	// This field is nullable.
	PublicUpdatesChannelID discord.ChannelID `json:"public_updates_channel_id,omitempty"`

	// PreferredLocale is the preferred locale of a "PUBLIC" guild used in
	// server discovery and notices from Discord.
	//
	// This defaults to "en-US".
	PreferredLocale option.NullableString `json:"preferred_locale,omitempty"`

	AuditLogReason `json:"-"`
}

// ModifyGuild modifies a guild's settings.
//
// Requires the MANAGE_GUILD permission.
//
// Fires a Guild Update Gateway event.
func (c *Client) ModifyGuild(id discord.GuildID, data ModifyGuildData) (*discord.Guild, error) {
	var g *discord.Guild
	return g, c.RequestJSON(
		&g, "PATCH",
		EndpointGuilds+id.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)

}

// DeleteGuild deletes a guild permanently. The User must be owner.
//
// Fires a Guild Delete Gateway event.
func (c *Client) DeleteGuild(id discord.GuildID) error {
	return c.FastRequest("DELETE", EndpointGuilds+id.String())
}

// VoiceRegionsGuild is the same as /voice, but returns VIP ones as well if
// available.
func (c *Client) VoiceRegionsGuild(guildID discord.GuildID) ([]discord.VoiceRegion, error) {
	var vrs []discord.VoiceRegion
	return vrs, c.RequestJSON(&vrs, "GET", EndpointGuilds+guildID.String()+"/regions")
}

// https://discord.com/developers/docs/resources/audit-log#get-guild-audit-log-query-string-parameters
type AuditLogData struct {
	// UserID filters the log for actions made by a user.
	UserID discord.UserID `schema:"user_id,omitempty"`
	// ActionType is the type of audit log event.
	ActionType discord.AuditLogEvent `schema:"action_type,omitempty"`
	// Before filters the log before a certain entry ID.
	Before discord.AuditLogEntryID `schema:"before,omitempty"`
	// Limit limits how many entries are returned (default 50, minimum 1,
	// maximum 100).
	Limit uint `schema:"limit"`
}

// AuditLog returns an audit log object for the guild.
//
// Requires the VIEW_AUDIT_LOG permission.
func (c *Client) AuditLog(guildID discord.GuildID, data AuditLogData) (*discord.AuditLog, error) {
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
func (c *Client) Integrations(guildID discord.GuildID) ([]discord.Integration, error) {

	var ints []discord.Integration
	return ints, c.RequestJSON(&ints, "GET", EndpointGuilds+guildID.String()+"/integrations")
}

// AttachIntegration attaches an integration object from the current user to
// the guild.
//
// Requires the MANAGE_GUILD permission.
//
// Fires a Guild Integrations Update Gateway event.
func (c *Client) AttachIntegration(
	guildID discord.GuildID,
	integrationID discord.IntegrationID, integrationType discord.Service) error {

	var param struct {
		Type discord.Service       `json:"type"`
		ID   discord.IntegrationID `json:"id"`
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
	guildID discord.GuildID,
	integrationID discord.IntegrationID, data ModifyIntegrationData) error {

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/integrations/"+integrationID.String(),
		httputil.WithJSONBody(data),
	)
}

// SyncIntegration syncs an integration.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) SyncIntegration(
	guildID discord.GuildID, integrationID discord.IntegrationID) error {

	return c.FastRequest(
		"POST",
		EndpointGuilds+guildID.String()+"/integrations/"+integrationID.String()+"/sync",
	)
}

// GuildWidgetSettings returns the guild widget object.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) GuildWidgetSettings(
	guildID discord.GuildID) (*discord.GuildWidgetSettings, error) {

	var ge *discord.GuildWidgetSettings
	return ge, c.RequestJSON(&ge, "GET", EndpointGuilds+guildID.String()+"/widget")
}

// ModifyGuildWidgetData is the structure to modify a guild widget object for
// the guild. All attributes may be passed in with JSON and modified.
//
// https://discord.com/developers/docs/resources/guild#guild-widget-object
type ModifyGuildWidgetData struct {
	// Enabled specifies whether the widget is enabled.
	Enabled option.Bool `json:"enabled,omitempty"`
	// ChannelID is the widget channel ID.
	ChannelID discord.ChannelID `json:"channel_id,omitempty"`

	AuditLogReason `json:"-"`
}

// ModifyGuildWidget modifies a guild widget object for the guild.
//
// Requires the MANAGE_GUILD permission.
func (c *Client) ModifyGuildWidget(
	guildID discord.GuildID, data ModifyGuildWidgetData) (*discord.GuildWidgetSettings, error) {

	var w *discord.GuildWidgetSettings
	return w, c.RequestJSON(
		&w, "PATCH",
		EndpointGuilds+guildID.String()+"/widget",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// GuildWidget returns the widget for the guild.
func (c *Client) GuildWidget(guildID discord.GuildID) (*discord.GuildWidget, error) {
	var w *discord.GuildWidget
	return w, c.RequestJSON(
		&w, "GET",
		EndpointGuilds+guildID.String()+"/widget.json")
}

// GuildVanityInvite returns the vanity invite for guilds that have that
// feature enabled. Only Code and Uses are filled. Code will be "" if a vanity
// url for the guild is not set.
//
// Requires MANAGE_GUILD.
func (c *Client) GuildVanityInvite(guildID discord.GuildID) (*discord.Invite, error) {
	var inv *discord.Invite
	return inv, c.RequestJSON(&inv, "GET", EndpointGuilds+guildID.String()+"/vanity-url")
}

// https://discord.com/developers/docs/resources/guild#get-guild-widget-image-widget-style-options
type GuildWidgetImageStyle string

const (
	// GuildShield is a shield style widget with Discord icon and guild members
	// online count.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=shield
	GuildShield GuildWidgetImageStyle = "shield"
	// GuildBanner1 is a large image with guild icon, name and online count.
	// "POWERED BY DISCORD" as the footer of the widget.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner1
	GuildBanner1 GuildWidgetImageStyle = "banner1"
	// GuildBanner2 is a smaller widget style with guild icon, name and online
	// count. Split on the right with Discord logo.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner2
	GuildBanner2 GuildWidgetImageStyle = "banner2"
	// GuildBanner3 is a large image with guild icon, name and online count. In
	// the footer, Discord logo on the left and "Chat Now" on the right.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner3
	GuildBanner3 GuildWidgetImageStyle = "banner3"
	// GuildBanner4 is a large Discord logo at the top of the widget.
	// Guild icon, name and online count in the middle portion of the widget
	// and a "JOIN MY SERVER" button at the bottom.
	//
	// Example: https://discordapp.com/api/guilds/81384788765712384/widget.png?style=banner4
	GuildBanner4 GuildWidgetImageStyle = "banner4"
)

// GuildWidgetImageURL returns a link to the PNG image widget for the guild.
//
// Requires no permissions or authentication.
func (c *Client) GuildWidgetImageURL(guildID discord.GuildID, img GuildWidgetImageStyle) string {
	return EndpointGuilds + guildID.String() + "/widget.png?style=" + string(img)
}

// GuildWidgetImage returns a PNG image widget for the guild. Requires no permissions
// or authentication.
func (c *Client) GuildWidgetImage(
	guildID discord.GuildID, img GuildWidgetImageStyle) (io.ReadCloser, error) {

	r, err := c.Request("GET", c.GuildWidgetImageURL(guildID, img))
	if err != nil {
		return nil, err
	}

	return r.GetBody(), nil
}
