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
	// ID is the id of this channel.
	ID ChannelID `json:"id"`
	// GuildID is the id of the guild.
	//
	// This field may be missing for some channel objects received over gateway
	// guild dispatches.
	GuildID GuildID `json:"guild_id,omitempty"`

	// Type is the type of channel.
	Type ChannelType `json:"type,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	NSFW bool `json:"nsfw,omitempty"`

	// Position is the sorting position of the channel.
	Position int `json:"position,omitempty"`
	// Overwrites are the explicit permission overrides for members
	// and roles.
	Overwrites []Overwrite `json:"permission_overwrites,omitempty"`

	// Name is the name of the channel (2-100 characters).
	Name string `json:"name,omitempty"`
	// Topic is the channel topic (0-1024 characters).
	Topic string `json:"topic,omitempty"`

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

	// DMRecipients are the recipients of the DM.
	DMRecipients []User `json:"recipients,omitempty"`
	// Icon is the icon hash.
	Icon Hash `json:"icon,omitempty"`

	// OwnerID is the id of the DM or thread creator.
	OwnerID UserID `json:"owner_id,omitempty"`
	// AppID is the application id of the group DM creator if it is
	// bot-created.
	AppID AppID `json:"application_id,omitempty"`
	// ParentID for guild channels: id of the parent category for a channel
	// (each parent category can contain up to 50 channels), for threads: the
	// id of the text channel this thread was created.
	ParentID ChannelID `json:"parent_id,omitempty"`

	// LastPinTime is when the last pinned message was pinned.
	LastPinTime Timestamp `json:"last_pin_timestamp,omitempty"`

	// RTCRegionID is the voice region id for the voice channel.
	RTCRegionID string `json:"rtc_region,omitempty"`
	// VideoQualityMode is the camera video quality mode of the voice channel.
	VideoQualityMode VideoQualityMode `json:"video_quality_mode,omitempty"`

	// MessageCount is an approximate count of messages in a thread. However,
	// counting stops at 50.
	MessageCount int `json:"message_count,omitempty"`
	// MemberCount is an approximate count of users in a thread. However,
	// counting stops at 50.
	MemberCount int `json:"member_count,omitempty"`

	// ThreadMetadata contains thread-specific fields not needed by other
	// channels.
	ThreadMetadata *ThreadMetadata `json:"thread_metadata,omitempty"`
	// ThreadMember is the thread member object for the current user, if they
	// have joined the thread, only included on certain API endpoints.
	ThreadMember *ThreadMember `json:"thread_member,omitempty"`
	// DefaultAutoArchiveDuration is the default duration for newly created
	// threads, in minutes, to automatically archive the thread after recent
	// activity.
	DefaultAutoArchiveDuration ArchiveDuration `json:"default_auto_archive_duration,omitempty"`

	// SelfPermissions are the computed permissions for the invoking user in
	// the channel, including overwrites, only included when part of the
	// resolved data received on a slash command interaction.
	SelfPermissions Permissions `json:"permissions,omitempty,string"`
}

func (ch *Channel) UnmarshalJSON(data []byte) error {
	type RawChannel Channel
	if err := json.Unmarshal(data, (*RawChannel)(ch)); err != nil {
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

// ThreadMetadata contains a number of thread-specific channel fields that are
// not needed by other channel types.
//
// https://discord.com/developers/docs/resources/channel#thread-metadata-object
type ThreadMetadata struct {
	// Archived specifies whether the thread is archived.
	Archived bool `json:"archived"`
	// AutoArchiveDuration is the duration in minutes to automatically archive
	// the thread after recent activity.
	AutoArchiveDuration ArchiveDuration `json:"auto_archive_duration"`
	// ArchiveTimestamp timestamp when the thread's archive status was last
	// changed, used for calculating recent activity.
	ArchiveTimestamp Timestamp `json:"archive_timestamp"`
	// Locked specifies whether the thread is locked; when a thread is locked,
	// only users with MANAGE_THREADS can unarchive it.
	Locked bool `json:"locked"`
	// Invitable specifies whether non-moderators can add other
	// non-moderators to a thread; only available on private threads.
	Invitable bool `json:"invitable,omitempty"`
}

type ThreadMember struct {
	// ID is the id of the thread.
	//
	// This field will be omitted on the member sent within each thread in the
	// guild create event.
	ID ChannelID `json:"id,omitempty"`
	// UserID is the id of the user.
	//
	// This field will be omitted on the member sent within each thread in the
	// guild create event.
	UserID UserID `json:"user_id,omitempty"`
	// Member is the member, only included in Thread Members Update Events.
	Member *Member `json:"member,omitempty"`
	// Presence is the presence, only included in Thread Members Update Events.
	Presence *Presence `json:"presence,omitempty"`
	// JoinTimestamp is the time the current user last joined the thread.
	JoinTimestamp Timestamp `json:"join_timestamp"`
	// Flags are any user-thread settings.
	Flags ThreadMemberFlags `json:"flags"`
}

// ThreadMemberFlags are the flags of a ThreadMember.
// Currently, none are documented.
type ThreadMemberFlags uint64
