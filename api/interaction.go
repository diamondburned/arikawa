package api

import (
	"mime/multipart"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

var EndpointInteractions = Endpoint + "interactions/"

type InteractionResponseType uint

// https://discord.com/developers/docs/interactions/slash-commands#interaction-response-interactioncallbacktype
const (
	PongInteraction InteractionResponseType = iota + 1
	_
	_
	MessageInteractionWithSource
	DeferredMessageInteractionWithSource
	DeferredMessageUpdate
	UpdateMessage
)

type InteractionResponse struct {
	Type InteractionResponseType  `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// InteractionResponseFlags implements flags for an InteractionApplicationCommandCallbackData.
// https://discord.com/developers/docs/interactions/slash-commands#interaction-response-interactionapplicationcommandcallbackdata
type InteractionResponseFlags uint

const EphemeralResponse InteractionResponseFlags = 64

// InteractionResponseData is InteractionApplicationCommandCallbackData in the
// official documentation.
type InteractionResponseData struct {
	TTS             option.NullableBool      `json:"tts,omitempty"`
	Content         option.NullableString    `json:"content,omitempty"`
	Components      *[]discord.Component     `json:"components,omitempty"`
	Embeds          *[]discord.Embed         `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions         `json:"allowed_mentions,omitempty"`
	Flags           InteractionResponseFlags `json:"flags,omitempty"`
	Files           []sendpart.File          `json:"-"`
}

// RespondInteraction responds to an incoming interaction. It is also known as
// an "interaction callback".
func (c *Client) RespondInteraction(
	id discord.InteractionID, token string, resp InteractionResponse) error {
	var URL = EndpointInteractions + id.String() + "/" + token + "/callback"
	return sendpart.POST(c.Client, resp, nil, URL)
}

// NeedsMultipart returns true if the InteractionResponse has files.
func (resp InteractionResponse) NeedsMultipart() bool {
	return resp.Data != nil && len(resp.Data.Files) > 0
}

func (resp InteractionResponse) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, resp, resp.Data.Files)
}
