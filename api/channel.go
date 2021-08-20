package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
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
	// Overwrites are the channel's permission overwrites.
	//
	// Channel Types: All
	Overwrites []discord.Overwrite `json:"permission_overwrites,omitempty"`
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

	AuditLogReason `json:"-"`
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
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

type (
	MoveChannelsData struct {
		// Channels are the channels to be moved.
		Channels []MoveChannelData

		AuditLogReason
	}

	MoveChannelData struct {
		// ID is the channel id.
		ID discord.ChannelID `json:"id"`
		// Position is the sorting position of the channel.
		Position option.Int `json:"position"`
		// LockPermissions syncs the permission overwrites with the new parent,
		// if moving to a new category.
		LockPermissions option.Bool `json:"lock_permissions"`
		// CategoryID is the new parent ID for the channel that is moved.
		CategoryID discord.ChannelID `json:"parent_id"`
	}
)

// MoveChannels modifies the position of channels in the guild.
//
// Requires MANAGE_CHANNELS.
//
// Fires multiple Channel Update Gateway events.
func (c *Client) MoveChannels(guildID discord.GuildID, data MoveChannelsData) error {
	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/channels",
		httputil.WithJSONBody(data.Channels), httputil.WithHeaders(data.Header()),
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
	// Position is the position of the channel in the left-hand listing.
	//
	// Channel Types: Text, News, Voice, Store, Category
	Position option.NullableInt `json:"position,omitempty"`
	// Topic is the 0-1024 character channel topic.
	//
	// Channel Types: Text, News
	Topic option.NullableString `json:"topic,omitempty"`
	// NSFW specifies whether the channel is nsfw.
	//
	// Channel Types: Text, News, Store
	NSFW option.NullableBool `json:"nsfw,omitempty"`
	// UserRateLimit is the amount of seconds a user has to wait before sending
	// another message (0-21600).
	// Bots, as well as users with the permission manage_messages or
	// manage_channel, are unaffected.
	//
	// Channel Types: Text, Thread
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
	// Overwrites are the channel or category-specific permissions.
	//
	// Channel Types: Text, News, Store, Voice, Category
	Overwrites *[]discord.Overwrite `json:"permission_overwrites,omitempty"`
	// CategoryID is the id of the new parent category for a channel.
	//
	// Channel Types: Text, News, Store, Voice
	CategoryID discord.ChannelID `json:"parent_id,string,omitempty"`

	// Icon is a base64 encoded icon.
	//
	// Channel Types: Group DM
	Icon string `json:"icon,omitempty"`

	// Archived specifies whether the thread is archived.
	Archived option.Bool `json:"archived,omitempty"`
	// AutoArchiveDuration is the duration in minutes to automatically archive
	// the thread after recent activity.
	//
	// Note that the three and seven day archive durations require the server
	// to be boosted.
	AutoArchiveDuration discord.ArchiveDuration `json:"auto_archive_duration,omitempty"`
	// Locked specifies whether the thread is locked. When a thread is locked,
	// only users with MANAGE_THREADS can unarchive it.
	Locked option.Bool `json:"locked,omitempty"`
	// Invitable specifies whether non-moderators can add other
	// non-moderators to a thread; only available on private threads
	Invitable option.Bool `json:"invitable,omitempty"`

	AuditLogReason `json:"-"`
}

// ModifyChannel updates a channel's settings.
//
// If modifying a guild channel, requires the MANAGE_CHANNELS permission for
// that guild. If modifying a thread, requires the MANAGE_THREADS permission.
// Furthermore, if modifying permission overwrites, the MANAGE_ROLES permission
// is required. Only permissions your bot has in the guild or channel can be
// allowed/denied (unless your bot has a MANAGE_ROLES overwrite in the
// channel).
//
// Fires a Channel Update event when modifying a guild channel, and a Thread
// Update event when modifying a thread.
func (c *Client) ModifyChannel(channelID discord.ChannelID, data ModifyChannelData) error {
	return c.FastRequest(
		"PATCH", EndpointChannels+channelID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// DeleteChannel deletes a channel, or closes a private message. Requires the
// MANAGE_CHANNELS permission for the guild. Deleting a category does not
// delete its child channels: they will have their parent_id removed and a
// Channel Update Gateway event will fire for each of them.
//
// Fires a Channel Delete Gateway event.
func (c *Client) DeleteChannel(
	channelID discord.ChannelID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointChannels+channelID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}

// https://discord.com/developers/docs/resources/channel#edit-channel-permissions-json-params
type EditChannelPermissionData struct {
	// Type is either "role" or "member".
	Type discord.OverwriteType `json:"type"`
	// Allow is a permission bit set for granted permissions.
	Allow discord.Permissions `json:"allow,string"`
	// Deny is a permission bit set for denied permissions.
	Deny discord.Permissions `json:"deny,string"`

	AuditLogReason `json:"-"`
}

// EditChannelPermission edits the channel's permission overwrites for a user
// or role in a channel. Only usable for guild channels.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) EditChannelPermission(
	channelID discord.ChannelID,
	overwriteID discord.Snowflake, data EditChannelPermissionData) error {

	return c.FastRequest(
		"PUT", EndpointChannels+channelID.String()+"/permissions/"+overwriteID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// DeleteChannelPermission deletes a channel permission overwrite for a user or
// role in a channel. Only usable for guild channels.
//
// Requires the MANAGE_ROLES permission.
func (c *Client) DeleteChannelPermission(
	channelID discord.ChannelID, overwriteID discord.Snowflake, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointChannels+channelID.String()+"/permissions/"+overwriteID.String(),
		httputil.WithHeaders(reason.Header()),
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
func (c *Client) PinMessage(
	channelID discord.ChannelID, messageID discord.MessageID, reason AuditLogReason) error {

	return c.FastRequest(
		"PUT", EndpointChannels+channelID.String()+"/pins/"+messageID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}

// UnpinMessage deletes a pinned message in a channel.
//
// Requires the MANAGE_MESSAGES permission.
func (c *Client) UnpinMessage(
	channelID discord.ChannelID, messageID discord.MessageID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointChannels+channelID.String()+"/pins/"+messageID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}

// AddRecipient adds a user to a group direct message. As accessToken is
// needed, clearly this endpoint should only be used for OAuth. AccessToken can
// be obtained with the "gdm.join" scope.
func (c *Client) AddRecipient(
	channelID discord.ChannelID, userID discord.UserID, accessToken, nickname string) error {

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
// write to the ack variable passed in. If this method is called
// asynchronously, then ack should be mutex guarded.
func (c *Client) Ack(channelID discord.ChannelID, messageID discord.MessageID, ack *Ack) error {
	return c.RequestJSON(
		ack, "POST",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+"/ack",
		httputil.WithJSONBody(ack),
	)
}

// https://discord.com/developers/docs/resources/channel#start-thread-with-message-json-params
// and
// https://discord.com/developers/docs/resources/channel#start-thread-without-message-json-params
type StartThreadData struct {
	// Name is the 1-100 character channel name.
	Name string `json:"name"`
	// AutoArchiveDuration is the duration in minutes to automatically archive
	// the thread after recent activity.
	//
	// Note that the three and seven day archive durations require the server
	// to be boosted.
	AutoArchiveDuration discord.ArchiveDuration `json:"auto_archive_duration"`
	// Type is the type of thread to create.
	//
	// This field can only be used when starting a thread without a message
	Type discord.ChannelType `json:"type,omitempty"` // we can omit, since thread types start at 10
	// Invitable specifies whether non-moderators can add other
	// non-moderators to a thread; only available on private threads.
	//
	// This field can only be used when starting a thread without a message
	Invitable bool `json:"invitable,omitempty"`

	AuditLogReason `json:"-"`
}

// StartThreadWithMessage creates a new thread from an existing message.
//
// When called on a GUILD_TEXT channel, creates a GUILD_PUBLIC_THREAD. When
// called on a GUILD_NEWS channel, creates a GUILD_NEWS_THREAD. The id of the
// created thread will be the same as the id of the message, and as such a
// message can only have a single thread created from it.
//
// Fires a Thread Create Gateway event.
func (c *Client) StartThreadWithMessage(
	channelID discord.ChannelID,
	messageID discord.MessageID, data StartThreadData) (*discord.Channel, error) {

	data.Type = 0

	var ch *discord.Channel
	return ch, c.RequestJSON(
		&ch, "POST",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+"/threads",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// StartThreadWithoutMessage creates a new thread that is not connected to an
// existing message.
//
// Fires a Thread Create Gateway event.
func (c *Client) StartThreadWithoutMessage(
	channelID discord.ChannelID, data StartThreadData) (*discord.Channel, error) {

	var ch *discord.Channel
	return ch, c.RequestJSON(
		&ch, "POST",
		EndpointChannels+channelID.String()+"/threads",
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// JoinThread adds the current user to a thread. Also requires the thread is
// not archived.
//
// Fires a Thread Members Update Gateway event.
func (c *Client) JoinThread(threadID discord.ChannelID) error {
	return c.FastRequest("PUT", EndpointChannels+threadID.String()+"/thread-members/@me")
}

// AddThreadMember adds another member to a thread. Requires the ability to
// send messages in the thread. Also requires the thread is not archived.
//
// Fires a Thread Members Update Gateway event.
func (c *Client) AddThreadMember(threadID discord.ChannelID, userID discord.UserID) error {
	return c.FastRequest(
		"PUT",
		EndpointChannels+threadID.String()+"/thread-members/"+userID.String(),
	)
}

// LeaveThread removes the current user from a thread. Also requires the thread
// is not archived.
//
// Fires a Thread Members Update Gateway event.
func (c *Client) LeaveThread(threadID discord.ChannelID) error {
	return c.FastRequest("DELETE", EndpointChannels+threadID.String()+"/thread-members/@me")
}

// RemoveThreadMember removes another member from a thread. Requires the
// MANAGE_THREADS permission, or the creator of the thread if it is a
// discord.GuildPrivateThread. Also requires the thread is not archived.
//
// Fires a Thread Members Update Gateway event.
func (c *Client) RemoveThreadMember(threadID discord.ChannelID, userID discord.UserID) error {
	return c.FastRequest(
		"DELETE",
		EndpointChannels+threadID.String()+"/thread-members/"+userID.String(),
	)
}

// ThreadMembers list all members of the thread.
//
// This endpoint is restricted according to whether the GUILD_MEMBERS
// Privileged Intent is enabled for your application.
func (c *Client) ThreadMembers(threadID discord.ChannelID) ([]discord.ThreadMember, error) {
	var m []discord.ThreadMember
	return m, c.RequestJSON(&m, "GET", EndpointChannels+threadID.String()+"/thread-members")
}

// https://discord.com/developers/docs/resources/guild#list-active-threads-response-body
type ActiveThreads struct {
	// Threads are the active threads, ordered by descending ID.
	Threads []discord.Channel `json:"threads"`
	// Members contains a thread member for each of the Threads the current
	// user has joined.
	Members []discord.ThreadMember `json:"members"`
}

// ActiveThreads returns all the active threads in the guild, including public
// and private threads.
func (c *Client) ActiveThreads(guildID discord.GuildID) (*ActiveThreads, error) {
	var t *ActiveThreads
	return t, c.RequestJSON(&t, "GET", EndpointGuilds+guildID.String()+"/threads/active")
}

// https://discord.com/developers/docs/resources/channel#list-public-archived-threads-response-body
// and
// https://discord.com/developers/docs/resources/channel#list-private-archived-threads-response-body
// and
// https://discord.com/developers/docs/resources/channel#list-private-archived-threads-response-body
type ArchivedThread struct {
	// Threads are the active threads, ordered by descending ArchiveTimestamp.
	Threads []discord.Channel `json:"threads"`
	// Members contains a thread member for each of the Threads the current
	// user has joined.
	Members []discord.ThreadMember `json:"members"`
	// More specifies whether there are potentially additional threads that
	// could be returned on a subsequent call.
	More bool `json:"has_more"`
}

// PublicArchivedThreadsBefore returns archived threads in the channel that are
// public.
//
// When called on a GUILD_TEXT channel, returns threads of type
// GUILD_PUBLIC_THREAD. When called on a GUILD_NEWS channel returns threads of
// type GUILD_NEWS_THREAD.
//
// Threads are ordered by ArchiveTimestamp, in descending order.
//
// Requires the READ_MESSAGE_HISTORY permission.
func (c *Client) PublicArchivedThreadsBefore(
	channelID discord.ChannelID,
	before discord.Timestamp, limit uint) ([]ArchivedThread, error) {

	var param struct {
		Before string `schema:"before,omitempty"`
		Limit  uint   `schema:"limit"`
	}

	if before.IsValid() {
		param.Before = before.Format(discord.TimestampFormat)
	}
	param.Limit = limit

	var t []ArchivedThread
	return t, c.RequestJSON(
		&t, "GET",
		EndpointChannels+channelID.String()+"/threads/archived/public",
		httputil.WithSchema(c, param),
	)
}

// PrivateArchivedThreadsBefore returns archived threads in the channel that
// are of type GUILD_PRIVATE_THREAD.
//
// Threads are ordered by ArchiveTimestamp, in descending order.
//
// Requires both the READ_MESSAGE_HISTORY and MANAGE_THREADS permissions.
func (c *Client) PrivateArchivedThreadsBefore(
	channelID discord.ChannelID,
	before discord.Timestamp, limit uint) ([]ArchivedThread, error) {

	var param struct {
		Before string `schema:"before,omitempty"`
		Limit  uint   `schema:"limit"`
	}

	if before.IsValid() {
		param.Before = before.Format(discord.TimestampFormat)
	}
	param.Limit = limit

	var t []ArchivedThread
	return t, c.RequestJSON(
		&t, "GET",
		EndpointChannels+channelID.String()+"/threads/archived/private",
		httputil.WithSchema(c, param),
	)
}

// JoinedPrivateArchivedThreadsBefore returns archived threads in the channel
// that are of type GUILD_PRIVATE_THREAD, and the user has joined.
//
// Threads are ordered by their ID, in descending order.
//
// Requires the READ_MESSAGE_HISTORY permission
func (c *Client) JoinedPrivateArchivedThreadsBefore(
	channelID discord.ChannelID,
	before discord.Timestamp, limit uint) ([]ArchivedThread, error) {

	var param struct {
		Before string `schema:"before,omitempty"`
		Limit  uint   `schema:"limit"`
	}

	if before.IsValid() {
		param.Before = before.Format(discord.TimestampFormat)
	}
	param.Limit = limit

	var t []ArchivedThread
	return t, c.RequestJSON(
		&t, "GET",
		EndpointChannels+channelID.String()+"/users/@me/threads/archived/private",
		httputil.WithSchema(c, param),
	)
}
