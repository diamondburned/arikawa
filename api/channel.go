package api

import (
	"git.sr.ht/~diamondburned/arikawa/discord"
	"git.sr.ht/~diamondburned/arikawa/httputil"
	"git.sr.ht/~diamondburned/arikawa/json"
)

const EndpointChannels = Endpoint + "channels/"

type Channel struct {
	ID   discord.Snowflake `json:"id,string"`
	Type ChannelType       `json:"type"`

	// Fields below may not appear

	GuildID discord.Snowflake `json:"guild_id,string,omitempty"`

	Position int    `json:"position,omitempty"`
	Name     string `json:"name,omitempty"`  // 2-100 chars
	Topic    string `json:"topic,omitempty"` // 0-1024 chars
	NSFW     bool   `json:"nsfw"`

	Icon discord.Hash `json:"icon,omitempty"`

	// Direct Messaging fields
	DMOwnerID    discord.Snowflake `json:"owner_id,omitempty"`
	DMRecipients []User            `json:"recipients,omitempty"`

	// AppID of the group DM creator if it's bot-created
	AppID discord.Snowflake `json:"application_id,omitempty"`

	// ID of the category the channel is in, if any.
	CategoryID discord.Snowflake `json:"parent_id,omitempty"`

	LastPinTime discord.Timestamp `json:"last_pin_timestamp,omitempty"`

	// Explicit permission overrides for members and roles.
	Permissions []Overwrite `json:"permission_overwrites,omitempty"`
	// ID of the last message, may not point to a valid one.
	LastMessageID discord.Snowflake `json:"last_message_id,omitempty"`

	// Slow mode duration. Bots and people with "manage_messages" or
	// "manage_channel" permissions are unaffected.
	UserRateLimit discord.Seconds `json:"rate_limit_per_user,omitempty"`

	// Voice, so GuildVoice only
	VoiceBitrate   int `json:"bitrate,omitempty"`
	VoiceUserLimit int `json:"user_limit,omitempty"`
}

type ChannelType uint8

const (
	GuildText ChannelType = iota
	DirectMessage
	GuildVoice
	GroupDM
	GuildCategory
	GuildNews
	GuildStore
)

type Overwrite struct {
	ID    discord.Snowflake `json:"id,omitempty"`
	Type  OverwriteType     `json:"type"`
	Allow uint64            `json:"allow"`
	Deny  uint64            `json:"deny"`
}

type OverwriteType string

const (
	OverwriteRole   OverwriteType = "role"
	OverwriteMember OverwriteType = "member"
)

type ChannelModifier struct {
	ChannelID discord.Snowflake `json:"id,omitempty"`

	// All types
	Name        string         `json:"name,omitempty"`
	Position    json.OptionInt `json:"position,omitempty"`
	Permissions []Overwrite    `json:"permission_overwrites,omitempty"`

	// Text only
	Topic json.OptionString `json:"topic,omitempty"`
	NSFW  json.OptionBool   `json:"nsfw,omitempty"`

	// 0-21600, refer to Channel.UserRateLimit
	UserRateLimit discord.Seconds `json:"rate_limit_per_user,omitempty"`

	// Voice only
	// 8000 - 96000 (or 128000 for Nitro)
	Bitrate json.OptionUint `json:"bitrate,omitempty"`
	// 0 no limit, 1-99
	UserLimit json.OptionUint `json:"user_limit,omitempty"`

	// Text OR Voice
	ParentID discord.Snowflake `json:"parent_id,omitempty"`
}

func (c *Client) Channel(channelID discord.Snowflake) (*Channel, error) {
	var channel *Channel

	return channel,
		c.RequestJSON(&channel, "POST", EndpointChannels+channelID.String())
}

func (c *Client) EditChannel(mod ChannelModifier) error {
	url := EndpointChannels + mod.ChannelID.String()
	mod.ChannelID = 0

	return c.FastRequest("PATCH", url, httputil.WithJSONBody(c, mod))
}

func (c *Client) DeleteChannel(channelID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String())
}

func (c *Client) EditChannelPermission(channelID discord.Snowflake,
	overwrite Overwrite) error {

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
	channelID discord.Snowflake) ([]Message, error) {

	var pinned []Message
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
func (c *Client) AddRecipient(channelID, userID discord.Snowflake,
	accessToken, nickname string) error {

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
