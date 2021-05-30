package api

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json/option"
)

var EndpointInteractions = Endpoint + "interactions/"

type InteractionResponseType uint

const (
	PongInteraction InteractionResponseType = iota + 1
	AcknowledgeInteraction
	MessageInteraction
	MessageInteractionWithSource
	AcknowledgeInteractionWithSource
	DeferredMessageUpdate
	UpdateMessage
)

type InteractionResponse struct {
	Type InteractionResponseType  `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// InteractionResponseData is InteractionApplicationCommandCallbackData in the
// official documentation.
type InteractionResponseData struct {
	TTS             option.NullableBool   `json:"tts,omitempty"`
	Content         option.NullableString `json:"content,omitempty"`
	Components      *[]discord.Component  `json:"components,omitempty"`
	Embeds          *[]discord.Embed      `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions      `json:"allowed_mentions,omitempty"`
}

// RespondInteraction responds to an incoming interaction. It is also known as
// an "interaction callback".
func (c *Client) RespondInteraction(
	id discord.InteractionID, token string, data InteractionResponse) error {
	return c.FastRequest(
		"POST",
		EndpointInteractions+id.String()+"/"+token+"/callback",
		httputil.WithJSONBody(data),
	)
}
