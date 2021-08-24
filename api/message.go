package api

import (
	"mime/multipart"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/internal/intmath"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

const (
	// the limit of max messages per request, as imposed by Discord
	maxMessageFetchLimit = 100
	// maxMessageDeleteLimit is the limit of max message that can be deleted
	// per bulk delete request, as imposed by Discord.
	maxMessageDeleteLimit = 100
)

// Messages returns a slice filled with the most recent messages sent in the
// channel with the passed ID. The method automatically paginates until it
// reaches the passed limit, or, if the limit is set to 0, has fetched all
// messages in the channel.
//
// As the underlying endpoint is capped at a maximum of 100 messages per
// request, at maximum a total of limit/100 rounded up requests will be made,
// although they may be less, if no more messages are available.
//
// When fetching the messages, those with the highest ID, will be fetched
// first.
// The returned slice will be sorted from latest to oldest.
func (c *Client) Messages(channelID discord.ChannelID, limit uint) ([]discord.Message, error) {
	// Since before is 0 it will be omitted by the http lib, which in turn
	// will lead discord to send us the most recent messages without having to
	// specify a Snowflake.
	return c.MessagesBefore(channelID, 0, limit)
}

// MessagesAround returns messages around the ID, with a limit of 100.
func (c *Client) MessagesAround(
	channelID discord.ChannelID, around discord.MessageID, limit uint) ([]discord.Message, error) {

	return c.messagesRange(channelID, 0, 0, around, limit)
}

// MessagesBefore returns a slice filled with the messages sent in the channel
// with the passed id. The method automatically paginates until it reaches the
// passed limit, or, if the limit is set to 0, has fetched all messages in the
// channel with an id smaller than before.
//
// As the underlying endpoint has a maximum of 100 messages per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more messages are available.
//
// The returned slice will be sorted from latest to oldest.
func (c *Client) MessagesBefore(
	channelID discord.ChannelID, before discord.MessageID, limit uint) ([]discord.Message, error) {

	msgs := make([]discord.Message, 0, limit)

	fetch := uint(maxMessageFetchLimit)

	// Check if we are truly fetching unlimited messages to avoid confusion
	// later on, if the limit reaches 0.
	unlimited := limit == 0

	for limit > 0 || unlimited {
		if limit > 0 {
			// Only fetch as much as we need. Since limit gradually decreases,
			// we only need to fetch intmath.Min(fetch, limit).
			fetch = uint(intmath.Min(maxMessageFetchLimit, int(limit)))
			limit -= maxMessageFetchLimit
		}

		m, err := c.messagesRange(channelID, before, 0, 0, fetch)
		if err != nil {
			return msgs, err
		}
		// Append the older messages into the list of newer messages.
		msgs = append(msgs, m...)

		if len(m) < maxMessageFetchLimit {
			break
		}

		before = m[len(m)-1].ID
	}

	if len(msgs) == 0 {
		return nil, nil
	}

	return msgs, nil
}

// MessagesAfter returns a slice filled with the messages sent in the channel
// with the passed ID. The method automatically paginates until it reaches the
// passed limit, or, if the limit is set to 0, has fetched all messages in the
// channel with an id higher than after.
//
// As the underlying endpoint has a maximum of 100 messages per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more messages are available.
//
// The returned slice will be sorted from latest to oldest.
func (c *Client) MessagesAfter(
	channelID discord.ChannelID, after discord.MessageID, limit uint) ([]discord.Message, error) {

	// 0 is uint's zero value and will lead to the after param getting omitted,
	// which in turn will lead to the most recent messages being returned.
	// Setting this to 1 will prevent that.
	if after == 0 {
		after = 1
	}

	var msgs []discord.Message

	fetch := uint(maxMessageFetchLimit)

	// Check if we are truly fetching unlimited messages to avoid confusion
	// later on, if the limit reaches 0.
	unlimited := limit == 0

	for limit > 0 || unlimited {
		if limit > 0 {
			// Only fetch as much as we need. Since limit gradually decreases,
			// we only need to fetch intmath.Min(fetch, limit).
			fetch = uint(intmath.Min(maxMessageFetchLimit, int(limit)))
			limit -= maxMessageFetchLimit
		}

		m, err := c.messagesRange(channelID, 0, after, 0, fetch)
		if err != nil {
			return msgs, err
		}
		// Prepend the older messages into the newly-fetched messages list.
		msgs = append(m, msgs...)

		if len(m) < maxMessageFetchLimit {
			break
		}

		after = m[0].ID
	}

	if len(msgs) == 0 {
		return nil, nil
	}

	return msgs, nil
}

func (c *Client) messagesRange(
	channelID discord.ChannelID,
	before, after, around discord.MessageID, limit uint) ([]discord.Message, error) {

	switch {
	case limit == 0:
		limit = 50
	case limit > 100:
		limit = 100
	}

	var param struct {
		Before discord.MessageID `schema:"before,omitempty"`
		After  discord.MessageID `schema:"after,omitempty"`
		Around discord.MessageID `schema:"around,omitempty"`

		Limit uint `schema:"limit"`
	}

	param.Before = before
	param.After = after
	param.Around = around
	param.Limit = limit

	var msgs []discord.Message
	return msgs, c.RequestJSON(
		&msgs, "GET",
		EndpointChannels+channelID.String()+"/messages",
		httputil.WithSchema(c, param),
	)
}

// Message returns a specific message in the channel.
//
// If operating on a guild channel, this endpoint requires the
// READ_MESSAGE_HISTORY permission to be present on the current user.
func (c *Client) Message(
	channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {

	var msg *discord.Message
	return msg, c.RequestJSON(&msg, "GET",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String())
}

// SendTextReply posts a text-only reply to a message ID in a guild text or DM channel
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendTextReply(
	channelID discord.ChannelID,
	content string, referenceID discord.MessageID) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Content:   content,
		Reference: &discord.MessageReference{MessageID: referenceID},
	})
}

// SendEmbeds sends embeds to a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendEmbeds(
	channelID discord.ChannelID, e ...discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Embeds: e,
	})
}

// SendEmbedReply posts an Embed reply to a message ID in a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendEmbedReply(
	channelID discord.ChannelID,
	referenceID discord.MessageID, embeds ...discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Embeds:    embeds,
		Reference: &discord.MessageReference{MessageID: referenceID},
	})
}

// SendMessage posts a message to a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendMessage(
	channelID discord.ChannelID,
	content string, embeds ...discord.Embed) (*discord.Message, error) {

	data := SendMessageData{
		Content: content,
		Embeds:  embeds,
	}
	return c.SendMessageComplex(channelID, data)
}

// SendMessageReply posts a reply to a message ID in a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendMessageReply(
	channelID discord.ChannelID, content string,
	referenceID discord.MessageID, embeds ...discord.Embed) (*discord.Message, error) {

	data := SendMessageData{
		Content:   content,
		Reference: &discord.MessageReference{MessageID: referenceID},
		Embeds:    embeds,
	}

	return c.SendMessageComplex(channelID, data)
}

// https://discord.com/developers/docs/resources/channel#edit-message
type EditMessageData struct {
	// Content is the new message contents (up to 2000 characters).
	Content option.NullableString `json:"content,omitempty"`
	// Embeds contains embedded rich content.
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// Components contains the new components to attach.
	Components *[]discord.Component `json:"components,omitempty"`
	// AllowedMentions are the allowed mentions for a message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	// Attachments are the attached files to keep
	Attachments *[]discord.Attachment `json:"attachments,omitempty"`
	// Flags edits the flags of a message (only SUPPRESS_EMBEDS can currently
	// be set/unset)
	//
	// This field is nullable.
	Flags *discord.MessageFlags `json:"flags,omitempty"`

	Files []sendpart.File `json:"-"`
}

// NeedsMultipart returns true if the SendMessageData has files.
func (data EditMessageData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

func (data EditMessageData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// EditText edits the contents of a previously sent message. For more
// documentation, refer to EditMessageComplex.
func (c *Client) EditText(
	channelID discord.ChannelID,
	messageID discord.MessageID, content string) (*discord.Message, error) {

	return c.EditMessageComplex(channelID, messageID, EditMessageData{
		Content: option.NewNullableString(content),
	})
}

// EditEmbeds edits the embed of a previously sent message. For more
// documentation, refer to EditMessageComplex.
func (c *Client) EditEmbeds(
	channelID discord.ChannelID,
	messageID discord.MessageID, embeds ...discord.Embed) (*discord.Message, error) {

	return c.EditMessageComplex(channelID, messageID, EditMessageData{
		Embeds: &embeds,
	})
}

// EditMessage edits a previously sent message. If content or embeds are empty
// the original content or embed will remain untouched. This means EditMessage
// will only update, but not remove parts of the message.
//
// For more documentation, refer to EditMessageComplex.
func (c *Client) EditMessage(
	channelID discord.ChannelID, messageID discord.MessageID,
	content string, embeds ...discord.Embed) (*discord.Message, error) {

	var data EditMessageData

	if len(content) > 0 {
		data.Content = option.NewNullableString(content)
	}

	if len(embeds) > 0 {
		data.Embeds = &embeds
	}

	return c.EditMessageComplex(channelID, messageID, data)
}

// EditMessageComplex edits a previously sent message. The fields Content,
// Embed, AllowedMentions and Flags can be edited by the original message
// author. Other users can only edit flags and only if they have the
// MANAGE_MESSAGES permission in the corresponding channel. When specifying
// flags, ensure to include all previously set flags/bits in addition to ones
// that you are modifying. Only flags documented in EditMessageData may be
// modified by users (unsupported flag changes are currently ignored without
// error).
//
// Fires a Message Update Gateway event.
func (c *Client) EditMessageComplex(
	channelID discord.ChannelID,
	messageID discord.MessageID, data EditMessageData) (*discord.Message, error) {

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	if data.Embeds != nil {
		sum := 0
		for i, embed := range *data.Embeds {
			if err := embed.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error at "+strconv.Itoa(i))
			}
			sum += embed.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of all text in embeds"}
			}

			(*data.Embeds)[i] = embed // embed.Validate changes fields
		}
	}

	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		EndpointChannels+channelID.String()+"/messages/"+messageID.String())
}

// CrosspostMessage crossposts a message in a news channel to following channels.
// This endpoint requires the SEND_MESSAGES permission if the current user sent the message,
// or additionally the MANAGE_MESSAGES permission for all other messages.
func (c *Client) CrosspostMessage(
	channelID discord.ChannelID, messageID discord.MessageID) (*discord.Message, error) {

	var msg *discord.Message

	return msg, c.RequestJSON(
		&msg,
		"POST",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String()+"/crosspost",
	)
}

// DeleteMessage delete a message. If operating on a guild channel and trying
// to delete a message that was not sent by the current user, this endpoint
// requires the MANAGE_MESSAGES permission.
func (c *Client) DeleteMessage(
	channelID discord.ChannelID, messageID discord.MessageID, reason AuditLogReason) error {

	return c.FastRequest(
		"DELETE", EndpointChannels+channelID.String()+"/messages/"+messageID.String(),
		httputil.WithHeaders(reason.Header()))
}

// DeleteMessages deletes multiple messages in a single request. This endpoint
// can only be used on guild channels and requires the MANAGE_MESSAGES
// permission. This endpoint only works for bots.
//
// This endpoint will not delete messages older than 2 weeks, and will fail if
// any message provided is older than that or if any duplicate message IDs are
// provided.
//
// Because the underlying endpoint only supports a maximum of 100 message IDs
// per request, DeleteMessages will make a total of messageIDs/100 rounded up
// requests.
//
// Fires a Message Delete Bulk Gateway event.
func (c *Client) DeleteMessages(
	channelID discord.ChannelID, messageIDs []discord.MessageID, reason AuditLogReason) error {

	switch {
	case len(messageIDs) == 0:
		return nil
	case len(messageIDs) == 1:
		return c.DeleteMessage(channelID, messageIDs[0], reason)
	case len(messageIDs) <= maxMessageDeleteLimit: // Fast path
		return c.deleteMessages(channelID, messageIDs, reason)
	}

	// If the number of messages to be deleted exceeds the amount discord is willing
	// to accept at one time then batches of messages will be deleted
	for start := 0; start < len(messageIDs); start += maxMessageDeleteLimit {
		end := intmath.Min(len(messageIDs), start+maxMessageDeleteLimit)
		err := c.deleteMessages(channelID, messageIDs[start:end], reason)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) deleteMessages(
	channelID discord.ChannelID, messageIDs []discord.MessageID, reason AuditLogReason) error {

	var param struct {
		Messages []discord.MessageID `json:"messages"`
	}

	param.Messages = messageIDs

	return c.FastRequest(
		"POST",
		EndpointChannels+channelID.String()+"/messages/bulk-delete",
		httputil.WithJSONBody(param), httputil.WithHeaders(reason.Header()),
	)
}
