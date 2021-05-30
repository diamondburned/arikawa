package api

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
)

var EndpointChannels = Endpoint + "channels/"

// Channels returns a list of guild channel objects.
func (c *Client) Channels(guildID discord.GuildID) ([]discord.Channel, error) {
	var chs []discord.Channel
	return chs, c.RequestJSON(&chs, "GET", EndpointGuilds+guildID.String()+"/channels")
}

// https://discord.com/developers/docs/resources/guild#create-guild-channel-json-params
type CreateChannelData struct {
	// Name is the channel name (2-100 characters).
	//
	// Channel Type: All
	Name string `json:"name"`
	// Type is the type of channel.
	//
	// Channel Type: All
	Type discord.ChannelType `json:"type,omitempty"`
	// Topic is the channel topic (0-1024 characters).
	//
	// Channel Types: Text, News
	Topic string `json:"topic,omitempty"`
	// VoiceBitrate is the bitrate (in bits) of the voice channel.
	// 8000 to 96000 (128000 for VIP servers)
	//
	// Channel Types: Voice
	VoiceBitrate uint `json:"bitrate,omitempty"`
	// VoiceUserLimit is the user limit of the voice channel.
	// 0 refers to no limit, 1 to 99 refers to a user limit.
	//
	// Channel Types: Voice
	VoiceUserLimit uint `json:"user_limit,omitempty"`
	// UserRateLimit is the amount of seconds a user has to wait before sending
	// another message (0-21600).
	// Bots, as well as users with the permission manage_messages or
	// manage_channel, are unaffected.
	//
	// Channel Types: Text
	UserRateLimit discord.Seconds `json:"rate_limit_per_user,omitempty"`
	// Position is the sorting position of the channel.
	//
	// Channel Types: All
	Position option.Int `json:"position,omitempty"`
	// Permissions are the channel's permission overwrites.
	//
	// Channel Types: All
	Permissions []discord.Overwrite `json:"permission_overwrites,omitempty"`
	// CategoryID is the 	id of the parent category for a channel.
	//
	// Channel Types: Text, News, Store, Voice
	CategoryID discord.ChannelID `json:"parent_id,string,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	//
	// Channel Types: Text, News, Store
	NSFW bool `json:"nsfw,omitempty"`
	// RTCRegionID is the channel voice region id. It will be determined
	// automatically set, if omitted.
	//
	// Channel Types: Voice
	RTCRegionID string `json:"rtc_region,omitempty"`
	// VideoQualityMode is the camera video quality mode of the voice channel.
	// This defaults to discord.AutoVideoQuality, if not set.
	//
	// ChannelTypes: Voice
	VoiceQualityMode discord.VideoQualityMode `json:"voice_quality_mode,omitempty"`
}

// CreateChannel creates a new channel object for the guild.
//
// Requires the MANAGE_CHANNELS permission. If setting permission overwrites,
// only permissions your bot has in the guild can be allowed/denied. Setting
// MANAGE_ROLES permission in channels is only possible for guild
// administrators. Returns the new channel object on success.
//
// Fires a ChannelCreate Gateway event.
func (c *Client) CreateChannel(
	guildID discord.GuildID, data CreateChannelData) (*discord.Channel, error) {
	var ch *discord.Channel
	return ch, c.RequestJSON(
		&ch, "POST",
		EndpointGuilds+guildID.String()+"/channels",
		httputil.WithJSONBody(data),
	)
}

type MoveChannelData struct {
	// ID is the channel id.
	ID discord.ChannelID `json:"id"`
	// Position is the sorting position of the channel
	Position option.Int `json:"position"`
}

// MoveChannel modifies the position of channels in the guild.
//
// Requires MANAGE_CHANNELS.
func (c *Client) MoveChannel(guildID discord.GuildID, data []MoveChannelData) error {
	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/channels", httputil.WithJSONBody(data),
	)
}

// Channel gets a channel by ID. Returns a channel object.
func (c *Client) Channel(channelID discord.ChannelID) (*discord.Channel, error) {
	var channel *discord.Channel
	return channel, c.RequestJSON(&channel, "GET", EndpointChannels+channelID.String())
}

// https://discord.com/developers/docs/resources/channel#modify-channel-json-params
type ModifyChannelData struct {
	// Name is the 2-100 character channel name.
	//
	// Channel Types: All
	Name string `json:"name,omitempty"`
	// Type is the type of the channel.
	// Only conversion between text and news is supported and only in guilds
	// with the "NEWS" feature
	//
	// Channel Types: Text, News
	Type *discord.ChannelType `json:"type,omitempty"`
	// Postion is the position of the channel in the left-hand listing
	//
	// Channel Types: All
	Position option.NullableInt `json:"position,omitempty"`
	// Topic is the 0-1024 character channel topic.
	//
	// Channel Types: Text, News
	Topic option.NullableString `json:"topic,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	//
	// Channel Types: Text, News, Store.
	NSFW option.NullableBool `json:"nsfw,omitempty"`
	// UserRateLimit is the amount of seconds a user has to wait before sending
	// another message (0-21600).
	// Bots, as well as users with the permission manage_messages or
	// manage_channel, are unaffected.
	//
	// Channel Types: Text
	UserRateLimit option.NullableUint `json:"rate_limit_per_user,omitempty"`
	// VoiceBitrate is the bitrate (in bits) of the voice channel.
	// 8000 to 96000 (128000 for VIP servers)
	//
	// Channel Types: Voice
	VoiceBitrate option.NullableUint `json:"bitrate,omitempty"`
	// VoiceUserLimit is the user limit of the voice channel.
	// 0 refers to no limit, 1 to 99 refers to a user limit.
	//
	// Channel Types: Voice
	VoiceUserLimit option.NullableUint `json:"user_limit,omitempty"`
	// Permissions are the channel or category-specific permissions.
	//
	// Channel Types: All
	Permissions *[]discord.Overwrite `json:"permission_overwrites,omitempty"`
	// CategoryID is the id of the new parent category for a channel.
	// Channel Types: Text, News, Store, Voice
	CategoryID discord.ChannelID `json:"parent_id,string,omitempty"`
}

// ModifyChannel updates a channel's settings.
//
// Requires the MANAGE_CHANNELS permission for the guild.
func (c *Client) ModifyChannel(channelID discord.ChannelID, data ModifyChannelData) error {
	return c.FastRequest("PATCH", EndpointChannels+channelID.String(), httputil.WithJSONBody(data))
}

// DeleteChannel deletes a channel, or closes a private message. Requires the
// MANAGE_CHANNELS permission for the guild. Deleting a category does not
// delete its child channels: they will have their parent_id removed and a
// Channel Update Gateway event will fire for each of them.
//
// Fires a Channel Delete Gateway event.
func (c *Client) DeleteChannel(channelID discord.ChannelID) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String())
}

// https://discord.com/developers/docs/resources/channel#edit-channel-permissions-json-params
type EditChannelPermissionData struct {
	// Type is either "role" or "member".
	Type discord.OverwriteType `json:"type"`
	// Allow is a permission bit set for granted permissions.
	Allow discord.Permissions `json:"allow,string"`
	// Deny is a permission bit set for denied permissions.
	Deny discord.Permissions `json:"deny,string"`
}

// EditChannelPermission edits the channel's permission overwrites for a user
// or role in a channel. Only usable for guild channels.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) EditChannelPermission(
	channelID discord.ChannelID, overwriteID discord.Snowflake, data EditChannelPermissionData) error {

	return c.FastRequest(
		"PUT", EndpointChannels+channelID.String()+"/permissions/"+overwriteID.String(),
		httputil.WithJSONBody(data),
	)
}

// DeleteChannelPermission deletes a channel permission overwrite for a user or
// role in a channel. Only usable for guild channels.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) DeleteChannelPermission(channelID discord.ChannelID, overwriteID discord.Snowflake) error {
	return c.FastRequest(
		"DELETE",
		EndpointChannels+channelID.String()+"/permissions/"+overwriteID.String(),
	)
}

// Typing posts a typing indicator to the channel. Undocumented, but the client
// usually clears the typing indicator after 8-10 seconds (or after a message).
func (c *Client) Typing(channelID discord.ChannelID) error {
	return c.FastRequest("POST", EndpointChannels+channelID.String()+"/typing")
}

// PinnedMessages returns all pinned messages in the channel as an array of
// message objects.
func (c *Client) PinnedMessages(channelID discord.ChannelID) ([]discord.Message, error) {
	var pinned []discord.Message
	return pinned, c.RequestJSON(&pinned, "GET", EndpointChannels+channelID.String()+"/pins")
}

// PinMessage pins a message in a channel.
//
// Requires the MANAGE_MESSAGES permission.
func (c *Client) PinMessage(channelID discord.ChannelID, messageID discord.MessageID) error {
	return c.FastRequest("PUT", EndpointChannels+channelID.String()+"/pins/"+messageID.String())
}

// UnpinMessage deletes a pinned message in a channel.
//
// Requires the MANAGE_MESSAGES permission.
func (c *Client) UnpinMessage(channelID discord.ChannelID, messageID discord.MessageID) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String()+"/pins/"+messageID.String())
}

// AddRecipient adds a user to a group direct message. As accessToken is needed,
// clearly this endpoint should only be used for OAuth. AccessToken can be
// obtained with the "gdm.join" scope.
func (c *Client) AddRecipient(channelID discord.ChannelID, userID discord.UserID, accessToken, nickname string) error {

	var params struct {
		AccessToken string `json:"access_token"`
		Nickname    string `json:"nickname"`
	}

	params.AccessToken = accessToken
	params.Nickname = nickname

	return c.FastRequest(
		"PUT",
		EndpointChannels+channelID.String()+"/recipients/"+userID.String(),
		httputil.WithJSONBody(params),
	)
}

// RemoveRecipient removes a user from a group direct message.
func (c *Client) RemoveRecipient(channelID discord.ChannelID, userID discord.UserID) error {
	return c.FastRequest(
		"DELETE",
		EndpointChannels+channelID.String()+"/recipients/"+userID.String(),
	)
}

// Ack is the read state of a channel. This is undocumented.
type Ack struct {
	Token string `json:"token"`
}

// Ack marks the read state of a channel. This is undocumented. The method will
// write to the ack variable passed in. If this method is called asynchronously,
// then ack should be mutex guarded.
func (c *Client) Ack(channelID discord.ChannelID, messageID discord.MessageID, ack *Ack) error {
	return c.RequestJSON(
		ack, "POST",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+"/ack",
		httputil.WithJSONBody(ack),
	)
}
