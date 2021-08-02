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
	Content string `json:"content,omitempty"`

	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
	// Embeds contains embedded rich content.
	//
	// Required: one of content, file, embeds
	Embeds []discord.Embed `json:"embeds,omitempty"`

	// Components is the list of components (such as buttons) to be attached to
	// the message.
	Components []discord.Component `json:"components,omitempty"`

	// Files represents a list of files to upload. This will not be
	// JSON-encoded and will only be available through WriteMultipart.
	Files []sendpart.File          `json:"-"`
	Flags InteractionResponseFlags `json:"flags,omitempty"`

	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
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

	var URL = EndpointInteractions + id.String() + "/" + token + "/callback"
	return sendpart.POST(c.Client, resp, nil, URL)
}

// OriginalInteractionResponse returns the initial interaction response.
func (c *Client) OriginalInteractionResponse(
	appID discord.AppID, token string) (*discord.Message, error) {

	var m *discord.Message
	return m, c.RequestJSON(
		&m, "GET",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

type EditInteractionResponseData struct {
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

	Files []sendpart.File `json:"-"`
}

// NeedsMultipart returns true if the SendMessageData has files.
func (data EditInteractionResponseData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

func (data EditInteractionResponseData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// EditOriginalInteractionResponse edits the initial Interaction response.
func (c *Client) EditOriginalInteractionResponse(
	appID discord.AppID,
	token string, data EditInteractionResponseData,
) (*discord.Message, error) {

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	if data.Embeds != nil {
		sum := 0
		for _, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of text in embeds"}
			}
		}
	}

	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

// DeleteOriginalInteractionResponse deletes the initial interaction response.
func (c *Client) DeleteOriginalInteractionResponse(appID discord.AppID, token string) error {
	return c.FastRequest("DELETE",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/@original")
}

// CreateInteractionFollowup creates a followup message for an interaction.
func (c *Client) CreateInteractionFollowup(
	appID discord.AppID, token string, data InteractionResponseData) (*discord.Message, error) {

	if data.Content == "" && len(data.Embeds) == 0 && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
	}

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	sum := 0
	for i, embed := range data.Embeds {
		if err := embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "embed error at "+strconv.Itoa(i))
		}
		sum += embed.Length()
		if sum > 6000 {
			return nil, &discord.OverboundError{sum, 6000, "sum of all text in embeds"}
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
		for _, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of text in embeds"}
			}
		}
	}

	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		EndpointWebhooks+appID.String()+"/"+token+"/messages/"+messageID.String())
}

// DeleteInteractionFollowup deletes a followup message for an interaction
func (c *Client) DeleteInteractionFollowup(
	appID discord.AppID, messageID discord.MessageID, token string) error {
	return c.FastRequest("DELETE",
		EndpointWebhooks+appID.String()+"/"+token+"/messages/"+messageID.String())
}
