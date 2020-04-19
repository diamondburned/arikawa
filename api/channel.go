package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json"
)

var EndpointChannels = Endpoint + "channels/"

func (c *Client) Channels(
	guildID discord.Snowflake) ([]discord.Channel, error) {

	var chs []discord.Channel
	return chs, c.RequestJSON(&chs, "GET",
		EndpointGuilds+guildID.String()+"/channels")
}

type CreateChannelData struct {
	Name  string `json:"name"` // 2-100
	Topic string `json:"topic,omitempty"`

	Type discord.ChannelType `json:"type,omitempty"`

	VoiceBitrate   uint `json:"bitrate,omitempty"`
	VoiceUserLimit uint `json:"user_limit,omitempty"`

	UserRateLimit discord.Seconds `json:"rate_limit_per_user,omitempty"`

	NSFW     bool `json:"nsfw"`
	Position int  `json:"position,omitempty"`

	Permissions []discord.Overwrite `json:"permission_overwrites,omitempty"`
	CategoryID  discord.Snowflake   `json:"parent_id,string,omitempty"`
}

func (c *Client) CreateChannel(
	guildID discord.Snowflake,
	data CreateChannelData) (*discord.Channel, error) {

	var ch *discord.Channel
	return ch, c.RequestJSON(
		&ch, "POST",
		EndpointGuilds+guildID.String()+"/channels",
		httputil.WithJSONBody(c, data),
	)
}

func (c *Client) MoveChannel(
	guildID, channelID discord.Snowflake, position int) error {

	var param struct {
		ID  discord.Snowflake `json:"id"`
		Pos int               `json:"position"`
	}

	param.ID = channelID
	param.Pos = position

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/channels",
		httputil.WithJSONBody(c, param),
	)
}

func (c *Client) Channel(
	channelID discord.Snowflake) (*discord.Channel, error) {

	var channel *discord.Channel

	return channel,
		c.RequestJSON(&channel, "GET", EndpointChannels+channelID.String())
}

type ModifyChannelData struct {
	// All types
	Name        string              `json:"name,omitempty"`
	Position    json.OptionInt      `json:"position,omitempty"`
	Permissions []discord.Overwrite `json:"permission_overwrites,omitempty"`

	// Text only
	Topic json.OptionString `json:"topic,omitempty"`
	NSFW  json.OptionBool   `json:"nsfw,omitempty"`

	// 0-21600 seconds, refer to (discord.Channel).UserRateLimit.
	UserRateLimit json.OptionInt `json:"rate_limit_per_user,omitempty"`

	// Voice only
	// 8000 - 96000 (or 128000 for Nitro)
	VoiceBitrate json.OptionUint `json:"bitrate,omitempty"`
	// 0 no limit, 1-99
	VoiceUserLimit json.OptionUint `json:"user_limit,omitempty"`

	// Text OR Voice
	CategoryID discord.Snowflake `json:"parent_id,string,omitempty"`
}

func (c *Client) ModifyChannel(channelID discord.Snowflake, data ModifyChannelData) error {
	return c.FastRequest(
		"PATCH",
		EndpointChannels+channelID.String(),
		httputil.WithJSONBody(c, data),
	)
}

func (c *Client) DeleteChannel(channelID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String())
}

func (c *Client) EditChannelPermission(
	channelID discord.Snowflake, overwrite discord.Overwrite) error {

	url := EndpointChannels + channelID.String() + "/permissions/" +
		overwrite.ID.String()
	overwrite.ID = 0

	return c.FastRequest("PUT", url, httputil.WithJSONBody(c, overwrite))
}

func (c *Client) DeleteChannelPermission(
	channelID, overwriteID discord.Snowflake) error {

	return c.FastRequest("DELETE", EndpointChannels+channelID.String()+
		"/permissions/"+overwriteID.String())
}

// Typing posts a typing indicator to the channel. Undocumented, but the client
// usually clears the typing indicator after 8-10 seconds (or after a message).
func (c *Client) Typing(channelID discord.Snowflake) error {
	return c.FastRequest("POST",
		EndpointChannels+channelID.String()+"/typing")
}

func (c *Client) PinnedMessages(
	channelID discord.Snowflake) ([]discord.Message, error) {

	var pinned []discord.Message
	return pinned, c.RequestJSON(&pinned, "GET",
		EndpointChannels+channelID.String()+"/pins")
}

// PinMessage pins a message, which requires MANAGE_MESSAGES/
func (c *Client) PinMessage(channelID, messageID discord.Snowflake) error {
	return c.FastRequest("PUT",
		EndpointChannels+channelID.String()+"/pins/"+messageID.String())
}

// UnpinMessage also requires MANAGE_MESSAGES.
func (c *Client) UnpinMessage(channelID, messageID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointChannels+channelID.String()+"/pins/"+messageID.String())
}

// AddRecipient adds a user to a group direct message. As accessToken is needed,
// clearly this endpoint should only be used for OAuth. AccessToken can be
// obtained with the "gdm.join" scope.
func (c *Client) AddRecipient(
	channelID, userID discord.Snowflake, accessToken, nickname string) error {

	var params struct {
		AccessToken string `json:"access_token"`
		Nickname    string `json:"nickname"`
	}

	params.AccessToken = accessToken
	params.Nickname = nickname

	return c.FastRequest(
		"PUT",
		EndpointChannels+channelID.String()+"/recipients/"+userID.String(),
		httputil.WithJSONBody(c, params),
	)
}

// RemoveRecipient removes a user from a group direct message.
func (c *Client) RemoveRecipient(channelID, userID discord.Snowflake) error {
	return c.FastRequest("DELETE",
		EndpointChannels+channelID.String()+"/recipients/"+userID.String())
}

// ACk is the read state of a channel. This is undocumented.
type Ack struct {
	Token string `json:"token"`
}

// Ack marks the read state of a channel. This is undocumented. The method will
// write to the ack variable passed in.
func (c *Client) Ack(channelID, messageID discord.Snowflake, ack *Ack) error {
	return c.RequestJSON(
		ack, "POST",
		EndpointChannels+channelID.String()+
			"/messages/"+messageID.String()+"/ack",
		httputil.WithJSONBody(c, ack),
	)
}
