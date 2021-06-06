package discord

import (
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

// Channel represents a guild or DM channel within Discord.
//
// https://discord.com/developers/docs/resources/channel#channel-object
type Channel struct {
	// LastPinTime is when the last pinned message was pinned.
	LastPinTime Timestamp `json:"last_pin_timestamp,omitempty"`
	// Topic is the channel topic (0-1024 characters).
	Topic string `json:"topic,omitempty"`
	// Icon is the icon hash.
	Icon Hash `json:"icon,omitempty"`
	// RTCRegionID is the voice region id for the voice channel. If set to
	// null, the voice region is determined automatically.
	//
	// If RTCRegionID is an empty string, the region is automatically
	// determined.
	RTCRegionID string `json:"rtc_region,omitempty"`
	// Name is the name of the channel (2-100 characters).
	Name string `json:"name,omitempty"`
	// DMRecipients are the recipients of the DM.
	DMRecipients []User `json:"recipients,omitempty"`

	// Permissions are the explicit permission overrides for members and roles.
	Permissions []Overwrite `json:"permission_overwrites,omitempty"`
	// ID is the id of this channel.
	ID ChannelID `json:"id"`
	// LastMessageID is the id of the last message sent in this channel (may
	// not point to an existing or valid message).
	LastMessageID MessageID `json:"last_message_id,omitempty"`
	// VoiceBitrate is the bitrate (in bits) of the voice channel.
	VoiceBitrate uint `json:"bitrate,omitempty"`
	// VoiceUserLimit is the user limit of the voice channel.
	VoiceUserLimit uint `json:"user_limit,omitempty"`
	// UserRateLimit is the amount of seconds a user has to wait before sending
	// another message (0-21600). Bots, as well as users with the permission
	// manage_messages or manage_channel, are unaffected.
	UserRateLimit Seconds `json:"rate_limit_per_user,omitempty"`
	// Position is the sorting position of the channel.
	Position int `json:"position,omitempty"`
	// GuildID is the id of the guild.
	GuildID GuildID `json:"guild_id,omitempty"`
	// DMOwnerID is the id of the DM creator.
	DMOwnerID UserID `json:"owner_id,omitempty"`
	// AppID is the application id of the group DM creator if it is
	// bot-created.
	AppID AppID `json:"application_id,omitempty"`
	// CategoryID is the id of the parent category for a channel (each parent
	// category can contain up to 50 channels).
	CategoryID ChannelID `json:"parent_id,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	NSFW bool `json:"nsfw,omitempty"`
	// Type is the type of channel.
	Type ChannelType `json:"type,omitempty"`
	// VideoQualityMode is the camera video quality mode of the voice channel.
	VideoQualityMode VideoQualityMode `json:"video_quality_mode,omitempty"`
}

func (ch *Channel) UnmarshalJSON(data []byte) error {
	type RawChannel Channel

	var newCh RawChannel
	if err := json.Unmarshal(data, &newCh); err != nil {
		return err
	}

	// In the docs, Discord states that if VideoQualityMode is omitted, it is
	// actually 1 aka. AutoVideoQuality, and they just didn't bother to send
	// it.
	// Refer to:
	// https://discord.com/developers/docs/resources/channel#channel-object-channel-structure
	if ch.VideoQualityMode == 0 {
		ch.VideoQualityMode = 1
	}

	return nil
}

func (ch Channel) MarshalJSON() ([]byte, error) {
	type RawChannel Channel

	if ch.RTCRegionID != "" {
		return json.Marshal(RawChannel(ch))
	}

	marshalChannel := struct {
		// Remove the ",omitempty" flag, forcing RTCRegionID to be marshalled
		// as JSON null. See the doc of Channel.RTCRegionID for more
		// information.
		RTCRegionID *string `json:"rtc_region"`
		RawChannel
	}{
		RawChannel:  RawChannel(ch),
		RTCRegionID: nil,
	}

	return json.Marshal(marshalChannel)
}

// CreatedAt returns a time object representing when the channel was created.
func (ch Channel) CreatedAt() time.Time {
	return ch.ID.Time()
}

// Mention returns a mention of the channel.
func (ch Channel) Mention() string {
	return ch.ID.Mention()
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
const (
	// GuildText is a text channel within a server.
	GuildText ChannelType = iota
	// DirectMessage is a direct message between users.
	DirectMessage
	// GuildVoice is a voice channel within a server.
	GuildVoice
	// GroupDM is a direct message between multiple users.
	GroupDM
	// GuildCategory is an organizational category that contains up to 50
	// channels.
	GuildCategory
	// GuildNews is a channel that users can follow and crosspost into their
	// own server.
	GuildNews
	// GuildStore is a channel in which game developers can sell their game on
	// Discord.
	GuildStore
	_
	_
	_
	// GuildNewsThread is a temporary sub-channel within a GUILD_NEWS channel
	GuildNewsThread
	// GuildPublicThread is a temporary sub-channel within a GUILD_TEXT
	// channel.
	GuildPublicThread
	// GuildPrivateThread isa temporary sub-channel within a GUILD_TEXT channel
	// that is only viewable by those invited and those with the MANAGE_THREADS
	// permission.
	GuildPrivateThread
	// GuildStageVoice is a voice channel for hosting events with an audience.
	GuildStageVoice
)

// https://discord.com/developers/docs/resources/channel#overwrite-object
type Overwrite struct {
	// ID is the role or user id.
	ID Snowflake `json:"id"`
	// Type indicates the entity overwritten: role or member.
	Type OverwriteType `json:"type"`
	// Allow is a permission bit set for granted permissions.
	Allow Permissions `json:"allow,string"`
	// Deny is a permission bit set for denied permissions.
	Deny Permissions `json:"deny,string"`
}

// OverwriteType is an enumerated type to indicate the entity being overwritten:
// role or member
type OverwriteType uint8

const (
	// OverwriteRole is an overwrite for a role.
	OverwriteRole OverwriteType = iota
	// OverwriteMember is an overwrite for a member.
	OverwriteMember
)

// UnmarshalJSON unmarshalls both a string-quoted number and a regular number
// into OverwriteType. We need to do this because Discord is so bad that they
// can't even handle 1s and 0s properly.
func (otype *OverwriteType) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)

	// It has been observed that discord still uses the "legacy" string
	// overwrite types in at least the guild create event.
	// Therefore this string check.
	switch s {
	case "role":
		*otype = OverwriteRole
		return nil
	case "member":
		*otype = OverwriteMember
		return nil
	}

	u, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return err
	}

	*otype = OverwriteType(u)
	return nil
}

type VideoQualityMode uint8

// https://discord.com/developers/docs/resources/channel#channel-object-video-quality-modes
const (
	AutoVideoQuality VideoQualityMode = iota + 1
	FullVideoQuality
)
