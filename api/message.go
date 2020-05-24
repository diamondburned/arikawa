package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json/option"
)

// Messages returns a list of messages sent in the channel with the passed ID.
// This method automatically paginates until it reaches the passed limit, or,
// if the limit is set to 0, has fetched all guilds within the passed ange.
//
// As the underlying endpoint has a maximum of 100 messages per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more messages are available.
//
// When fetching the messages, those with the smallest ID will be fetched
// first.
func (c *Client) Messages(channelID discord.Snowflake, limit uint) ([]discord.Message, error) {
	return c.MessagesAfter(channelID, 0, limit)
}

// MessagesAround returns messages around the ID, with a limit of 100.
func (c *Client) MessagesAround(
	channelID, around discord.Snowflake, limit uint) ([]discord.Message, error) {

	return c.messagesRange(channelID, 0, 0, around, limit)
}

// MessagesBefore returns a list messages sent in the channel with the passed
// ID. This method automatically paginates until it reaches the passed limit,
// or, if the limit is set to 0, has fetched all guilds within the passed
// range.
//
// As the underlying endpoint has a maximum of 100 messages per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more messages are available.
func (c *Client) MessagesBefore(
	channelID, before discord.Snowflake, limit uint) ([]discord.Message, error) {

	var msgs []discord.Message

	// this is the limit of max messages per request, as imposed by Discord
	const hardLimit int = 100

	unlimited := limit == 0

	for fetch := uint(hardLimit); limit > 0 || unlimited; fetch = uint(hardLimit) {
		if limit > 0 {
			if fetch > limit {
				fetch = limit
			}
			limit -= fetch
		}

		m, err := c.messagesRange(channelID, before, 0, 0, fetch)
		if err != nil {
			return msgs, err
		}
		msgs = append(m, msgs...)

		if len(m) < hardLimit {
			break
		}

		before = m[0].ID
	}

	return msgs, nil
}

// MessagesAfter returns a list messages sent in the channel with the passed
// ID. This method automatically paginates until it reaches the passed limit,
// or, if the limit is set to 0, has fetched all guilds within the passed
// range.
//
// As the underlying endpoint has a maximum of 100 messages per request, at
// maximum a total of limit/100 rounded up requests will be made, although they
// may be less, if no more messages are available.
func (c *Client) MessagesAfter(
	channelID, after discord.Snowflake, limit uint) ([]discord.Message, error) {

	var msgs []discord.Message

	// this is the limit of max messages per request, as imposed by Discord
	const hardLimit int = 100

	unlimited := limit == 0

	for fetch := uint(hardLimit); limit > 0 || unlimited; fetch = uint(hardLimit) {
		if limit > 0 {
			if fetch > limit {
				fetch = limit
			}
			limit -= fetch
		}

		m, err := c.messagesRange(channelID, 0, after, 0, fetch)
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, m...)

		if len(m) < hardLimit {
			break
		}

		after = m[len(m)-1].ID
	}

	return msgs, nil
}

func (c *Client) messagesRange(
	channelID, before, after, around discord.Snowflake, limit uint) ([]discord.Message, error) {

	switch {
	case limit == 0:
		limit = 50
	case limit > 100:
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before,omitempty"`
		After  discord.Snowflake `schema:"after,omitempty"`
		Around discord.Snowflake `schema:"around,omitempty"`

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
func (c *Client) Message(channelID, messageID discord.Snowflake) (*discord.Message, error) {
	var msg *discord.Message
	return msg, c.RequestJSON(&msg, "GET",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String())
}

// SendText posts a only-text message to a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendText(channelID discord.Snowflake, content string) (*discord.Message, error) {
	return c.SendMessageComplex(channelID, SendMessageData{
		Content: content,
	})
}

// SendEmbed posts an Embed to a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendEmbed(
	channelID discord.Snowflake, e discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Embed: &e,
	})
}

// SendMessage posts a message to a guild text or DM channel.
//
// If operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user.
//
// Fires a Message Create Gateway event.
func (c *Client) SendMessage(
	channelID discord.Snowflake, content string, embed *discord.Embed) (*discord.Message, error) {

	return c.SendMessageComplex(channelID, SendMessageData{
		Content: content,
		Embed:   embed,
	})
}

// https://discord.com/developers/docs/resources/channel#edit-message-json-params
type EditMessageData struct {
	// Content is the new message contents (up to 2000 characters).
	Content option.NullableString `json:"content,omitempty"`
	// Embed contains embedded rich content.
	Embed *discord.Embed `json:"embed,omitempty"`
	// AllowedMentions are the allowed mentions for a message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	// Flags edits the flags of a message (only SUPPRESS_EMBEDS can currently
	// be set/unset)
	//
	// This field is nullable.
	Flags *discord.MessageFlags `json:"flags,omitempty"`
}

// EditText edits the contents of a previously sent message. For more
// documentation, refer to EditMessageComplex.
func (c *Client) EditText(
	channelID, messageID discord.Snowflake, content string) (*discord.Message, error) {

	return c.EditMessageComplex(channelID, messageID, EditMessageData{
		Content: option.NewNullableString(content),
	})
}

// EditEmbed edits the embed of a previously sent message. For more
// documentation, refer to EditMessageComplex.
func (c *Client) EditEmbed(
	channelID, messageID discord.Snowflake, embed discord.Embed) (*discord.Message, error) {

	return c.EditMessageComplex(channelID, messageID, EditMessageData{
		Embed: &embed,
	})
}

// EditMessage edits a previously sent message. For more documentation, refer to
// EditMessageComplex.
func (c *Client) EditMessage(
	channelID, messageID discord.Snowflake, content string,
	embed *discord.Embed, suppressEmbeds bool) (*discord.Message, error) {

	var data = EditMessageData{
		Content: option.NewNullableString(content),
		Embed:   embed,
	}
	if suppressEmbeds {
		data.Flags = &discord.SuppressEmbeds
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
	channelID, messageID discord.Snowflake, data EditMessageData) (*discord.Message, error) {

	var msg *discord.Message
	return msg, c.RequestJSON(
		&msg, "PATCH",
		EndpointChannels+channelID.String()+"/messages/"+messageID.String(),
		httputil.WithJSONBody(data),
	)
}

// DeleteMessage delete a message. If operating on a guild channel and trying
// to delete a message that was not sent by the current user, this endpoint
// requires the MANAGE_MESSAGES permission.
func (c *Client) DeleteMessage(channelID, messageID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointChannels+channelID.String()+
		"/messages/"+messageID.String())
}

// DeleteMessages deletes multiple messages in a single request. This endpoint
// can only be used on guild channels and requires the MANAGE_MESSAGES
// permission. This endpoint only works for bots.
//
// This endpoint will not delete messages older than 2 weeks, and will fail if
// any message provided is older than that or if any duplicate message IDs are
// provided.
//
// Fires a Message Delete Bulk Gateway event.
func (c *Client) DeleteMessages(channelID discord.Snowflake, messageIDs []discord.Snowflake) error {
	var param struct {
		Messages []discord.Snowflake `json:"messages"`
	}

	param.Messages = messageIDs

	return c.FastRequest(
		"POST",
		EndpointChannels+channelID.String()+"/messages/bulk-delete",
		httputil.WithJSONBody(param),
	)
}
