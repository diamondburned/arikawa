package discord

// https://discord.com/developers/docs/resources/channel#channel-object
type Channel struct {
	// ID is the id of this channel.
	ID Snowflake `json:"id,string"`
	// Type is the type of channel.
	Type ChannelType `json:"type"`
	// GuildID is the id of the guild.
	GuildID Snowflake `json:"guild_id,string,omitempty"`

	// Position is the sorting position of the channel.
	Position int `json:"position,omitempty"`
	// Permissions are the explicit permission overrides for members and roles.
	Permissions []Overwrite `json:"permission_overwrites,omitempty"`

	// Name is the name of the channel (2-100 characters).
	Name string `json:"name,omitempty"`
	// Topic is the channel topic (0-1024 characters).
	Topic string `json:"topic,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	NSFW bool `json:"nsfw"`

	// LastMessageID is the id of the last message sent in this channel (may
	// not point to an existing or valid message).
	LastMessageID Snowflake `json:"last_message_id,string,omitempty"`

	// VoiceBitrate is the bitrate (in bits) of the voice channel.
	VoiceBitrate uint `json:"bitrate,omitempty"`
	// VoiceUserLimit is the user limit of the voice channel.
	VoiceUserLimit uint `json:"user_limit,omitempty"`

	// UserRateLimit is the amount of seconds a user has to wait before sending
	// another message (0-21600). Bots, as well as users with the permission
	// manage_messages or manage_channel, are unaffected.
	UserRateLimit Seconds `json:"rate_limit_per_user,omitempty"`

	// DMRecipients are the recipients of the DM.
	DMRecipients []User `json:"recipients,omitempty"`
	// Icon is the icon hash.
	Icon Hash `json:"icon,omitempty"`
	// DMOwnerID is the id of the DM creator.
	DMOwnerID Snowflake `json:"owner_id,string,omitempty"`

	// AppID is the application id of the group DM creator if it is
	// bot-created.
	AppID Snowflake `json:"application_id,string,omitempty"`

	// CategoryID is the id of the parent category for a channel (each parent
	// category can contain up to 50 channels).
	CategoryID Snowflake `json:"parent_id,string,omitempty"`
	// LastPinTime is when the last pinned message was pinned.
	LastPinTime Timestamp `json:"last_pin_timestamp,omitempty"`
}

// Mention returns a mention of the channel.
func (ch Channel) Mention() string {
	return "<#" + ch.ID.String() + ">"
}

// IconURL returns the URL to the channel icon in the PNG format.
// An empty string is returned if there's no icon.
func (ch Channel) IconURL() string {
	return ch.IconURLWithType(PNGImage)
}

// IconURLWithType returns the URL to the channel icon using the passed
// ImageType. An empty string is returned if there's no icon.
//
// Supported ImageTypes: PNG, JPEG, WebP
func (ch Channel) IconURLWithType(t ImageType) string {
	if ch.Icon == "" {
		return ""
	}

	return "https://cdn.discordapp.com/channel-icons/" +
		ch.ID.String() + "/" + t.format(ch.Icon)
}

type ChannelType uint8

// https://discord.com/developers/docs/resources/channel#channel-object-channel-types
var (
	// GuildText is a text channel within a server.
	GuildText ChannelType = 0
	// DirectMessage is a direct message between users.
	DirectMessage ChannelType = 1
	// GuildVoice is a voice channel within a server.
	GuildVoice ChannelType = 2
	// GroupDM is a direct message between multiple users.
	GroupDM ChannelType = 3
	// GuildCategory is an organizational category that contains up to 50
	// channels.
	GuildCategory ChannelType = 4
	// GuildNews is a channel that users can follow and crosspost into their
	// own server.
	GuildNews ChannelType = 5
	// GuildStore is a channel in which game developers can sell their game on
	// Discord.
	GuildStore ChannelType = 6
)

// https://discord.com/developers/docs/resources/channel#overwrite-object
type Overwrite struct {
	// ID is the role or user id.
	ID Snowflake `json:"id,string"`
	// Type is either "role" or "member".
	Type OverwriteType `json:"type"`
	// Allow is a permission bit set for granted permissions.
	Allow Permissions `json:"allow"`
	// Deny is a permission bit set for denied permissions.
	Deny Permissions `json:"deny"`
}

type OverwriteType string

const (
	// OverwriteRole is an overwrite for a role.
	OverwriteRole OverwriteType = "role"
	// OverwriteMember is an overwrite for a member.
	OverwriteMember OverwriteType = "member"
)
