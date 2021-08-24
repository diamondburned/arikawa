package api

import (
	"mime/multipart"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

var EndpointInteractions = Endpoint + "interactions/"

type InteractionResponseType uint

// https://discord.com/developers/docs/interactions/slash-commands#interaction-response-object-interaction-callback-type
const (
	PongInteraction InteractionResponseType = iota + 1
	_
	_
	MessageInteractionWithSource
	DeferredMessageInteractionWithSource
	DeferredMessageUpdate
	UpdateMessage
)

// InteractionResponseFlags implements flags for an
// InteractionApplicationCommandCallbackData.
//
// https://discord.com/developers/docs/interactions/slash-commands#interaction-response-object-interaction-application-command-callback-data-flags
type InteractionResponseFlags uint

const EphemeralResponse InteractionResponseFlags = 64

type InteractionResponse struct {
	Type InteractionResponseType  `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// NeedsMultipart returns true if the InteractionResponse has files.
func (resp InteractionResponse) NeedsMultipart() bool {
	return resp.Data != nil && resp.Data.NeedsMultipart()
}

func (resp InteractionResponse) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, resp, resp.Data.Files)
}

// InteractionResponseData is InteractionApplicationCommandCallbackData in the
// official documentation.
type InteractionResponseData struct {
	// Content are the message contents (up to 2000 characters).
	//
	// Required: one of content, file, embeds
	Content option.NullableString `json:"content,omitempty"`
	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
	// Embeds contains embedded rich content.
	//
	// Required: one of content, file, embeds
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// Components is the list of components (such as buttons) to be attached to
	// the message.
	Components *[]discord.Component `json:"components,omitempty"`
	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	// Flags are the interaction application command callback data flags.
	Flags InteractionResponseFlags `json:"flags,omitempty"`

	// Files represents a list of files to upload. This will not be
	// JSON-encoded and will only be available through WriteMultipart.
	Files []sendpart.File `json:"-"`
}

// NeedsMultipart returns true if the InteractionResponseData has files.
func (d InteractionResponseData) NeedsMultipart() bool {
	return len(d.Files) > 0
}

func (d InteractionResponseData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, d, d.Files)
}

// RespondInteraction responds to an incoming interaction. It is also known as
// an "interaction callback".
func (c *Client) RespondInteraction(
	id discord.InteractionID, token string, resp InteractionResponse) error {

	if resp.Data != nil {
		if resp.Type == MessageInteractionWithSource {
			// A new message is being created, make sure none of the fields
			// are null or empty.
			if (resp.Data.Content == nil || resp.Data.Content.Val == "") &&
				(resp.Data.Embeds == nil || len(*resp.Data.Embeds) == 0) &&
				len(resp.Data.Files) == 0 {
				return ErrEmptyMessage
			}
		} else if resp.Type == UpdateMessage {
			// A component is being updated. We therefore don't know what
			// fields are filled. The only thing we can check is if content,
			// embeds and files are null.
			if (resp.Data.Content != nil && !resp.Data.Content.Init) &&
				(resp.Data.Embeds != nil && *resp.Data.Embeds == nil) && len(resp.Data.Files) == 0 {
				return ErrEmptyMessage
			}
		}

		if resp.Data.AllowedMentions != nil {
			if err := resp.Data.AllowedMentions.Verify(); err != nil {
				return errors.Wrap(err, "allowedMentions error")
			}
		}

		if resp.Data.Embeds != nil {
			sum := 0
			for i, embed := range *resp.Data.Embeds {
				if err := embed.Validate(); err != nil {
					return errors.Wrap(err, "embed error at "+strconv.Itoa(i))
				}
				sum += embed.Length()
				if sum > 6000 {
					return &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of all text in embeds"}
				}

				(*resp.Data.Embeds)[i] = embed // embed.Validate changes fields
			}
		}
	}

	var URL = EndpointInteractions + id.String() + "/" + token + "/callback"
	return sendpart.POST(c.Client, resp, nil, URL)
}

// InteractionResponse returns the initial interaction response.
func (c *Client) InteractionResponse(
	appID discord.AppID, token string) (*discord.Message, error) {

	var m *discord.Message
	return m, c.RequestJSON(
		&m, "GET",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

type EditInteractionResponseData struct {
	// Content are the new message contents (up to 2000 characters).
	Content option.NullableString `json:"content,omitempty"`
	// Embeds contains embedded rich content.
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// Components contains the new components to attach.
	Components *[]discord.Component `json:"components,omitempty"`
	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	// Attachments are the attached files to keep.
	Attachments *[]discord.Attachment `json:"attachments,omitempty"`

	// Files represents a list of files to upload. This will not be
	// JSON-encoded and will only be available through WriteMultipart.
	Files []sendpart.File `json:"-"`
}

// NeedsMultipart returns true if the SendMessageData has files.
func (data EditInteractionResponseData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

func (data EditInteractionResponseData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// EditInteractionResponse edits the initial Interaction response.
func (c *Client) EditInteractionResponse(
	appID discord.AppID,
	token string, data EditInteractionResponseData) (*discord.Message, error) {

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	if data.Embeds != nil {
		sum := 0
		for i, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of text in embeds"}
			}

			(*data.Embeds)[i] = e // e.Validate changes fields
		}
	}

	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

// DeleteInteractionResponse deletes the initial interaction response.
func (c *Client) DeleteInteractionResponse(appID discord.AppID, token string) error {
	return c.FastRequest("DELETE",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

// CreateInteractionFollowup creates a followup message for an interaction.
func (c *Client) CreateInteractionFollowup(
	appID discord.AppID, token string, data InteractionResponseData) (*discord.Message, error) {

	if (data.Content == nil || data.Content.Val == "") &&
		(data.Embeds == nil || len(*data.Embeds) == 0) && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
	}

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
	return msg, sendpart.POST(
		c.Client, data, msg, EndpointWebhooks+appID.String()+"/"+token+"?")
}

func (c *Client) EditInteractionFollowup(
	appID discord.AppID, messageID discord.MessageID,
	token string, data EditInteractionResponseData) (*discord.Message, error) {

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	if data.Embeds != nil {
		sum := 0
		for i, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of text in embeds"}
			}

			(*data.Embeds)[i] = e // e.Validate changes fields
		}
	}

	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		EndpointWebhooks+appID.String()+"/"+token+"/messages/"+messageID.String())
}

// DeleteInteractionFollowup deletes a followup message for an interaction.
func (c *Client) DeleteInteractionFollowup(
	appID discord.AppID, messageID discord.MessageID, token string) error {

	return c.FastRequest("DELETE",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/"+messageID.String())
}
