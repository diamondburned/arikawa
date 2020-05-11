package discord

type Channel struct {
	ID   Snowflake   `json:"id,string"`
	Type ChannelType `json:"type"`

	// Fields below may not appear

	GuildID Snowflake `json:"guild_id,string,omitempty"`

	Position int    `json:"position,omitempty"`
	Name     string `json:"name,omitempty"`  // 2-100 chars
	Topic    string `json:"topic,omitempty"` // 0-1024 chars
	NSFW     bool   `json:"nsfw"`

	Icon Hash `json:"icon,omitempty"`

	// Direct Messaging fields
	DMOwnerID    Snowflake `json:"owner_id,string,omitempty"`
	DMRecipients []User    `json:"recipients,omitempty"`

	// AppID of the group DM creator if it's bot-created
	AppID Snowflake `json:"application_id,string,omitempty"`

	// ID of the category the channel is in, if any.
	CategoryID Snowflake `json:"parent_id,string,omitempty"`

	LastPinTime Timestamp `json:"last_pin_timestamp,omitempty"`

	// Explicit permission overrides for members and roles.
	Permissions []Overwrite `json:"permission_overwrites,omitempty"`
	// ID of the last message, may not point to a valid one.
	LastMessageID Snowflake `json:"last_message_id,string,omitempty"`

	// Slow mode duration. Bots and people with "manage_messages" or
	// "manage_channel" permissions are unaffected.
	UserRateLimit Seconds `json:"rate_limit_per_user,omitempty"`

	// Voice, so GuildVoice only
	VoiceBitrate   uint `json:"bitrate,omitempty"`
	VoiceUserLimit uint `json:"user_limit,omitempty"`
}

func (ch Channel) Mention() string {
	return "<#" + ch.ID.String() + ">"
}

// IconURL returns the icon of the channel. This function will only return
// something if ch.Icon is not empty.
func (ch Channel) IconURL() string {
	if ch.Icon == "" {
		return ""
	}

	return "https://cdn.discordapp.com/channel-icons/" +
		ch.ID.String() + "/" + ch.Icon + ".png"
}

type ChannelType uint8

var (
	GuildText     ChannelType = 0
	DirectMessage ChannelType = 1
	GuildVoice    ChannelType = 2
	GroupDM       ChannelType = 3
	GuildCategory ChannelType = 4
	GuildNews     ChannelType = 5
	GuildStore    ChannelType = 6
)

type Overwrite struct {
	ID    Snowflake     `json:"id,string,omitempty"`
	Type  OverwriteType `json:"type"`
	Allow Permissions   `json:"allow"`
	Deny  Permissions   `json:"deny"`
}

type OverwriteType string

const (
	OverwriteRole   OverwriteType = "role"
	OverwriteMember OverwriteType = "member"
)
