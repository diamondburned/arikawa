package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

// SendInteractionData is an interaction that a user sends to an application,
// such as when they click a button on the application's message. It is
// undocumented.
//
// https://docs.discord.food/interactions/receiving-and-responding#create-interaction
type SendInteractionData struct {
	// Type is either discord.ComponentInteractionType or
	// discord.ModalInteractionType.
	Type discord.InteractionDataType `json:"type"`
	// Nonce is echoed back in the gateway events about this interaction. It
	// is usually a snowflake.
	Nonce string `json:"nonce,omitempty"`
	// SessionID is the ID of the user's gateway session, which receives the
	// events about this interaction.
	SessionID string `json:"session_id"`

	AppID     discord.AppID     `json:"application_id"`
	GuildID   discord.GuildID   `json:"guild_id,omitempty"`
	ChannelID discord.ChannelID `json:"channel_id"`

	// MessageID and MessageFlags are those of the message that the component
	// is on. They are only used for component interactions.
	MessageID    discord.MessageID    `json:"message_id,omitempty"`
	MessageFlags discord.MessageFlags `json:"message_flags,omitempty"`

	// Data is either a ComponentInteractionData or a ModalInteractionData.
	Data any `json:"data"`
}

// ComponentInteractionData is the data of a user's interaction with a
// message component.
//
// https://docs.discord.food/interactions/receiving-and-responding#sendable-message-component-data-structure
type ComponentInteractionData struct {
	ComponentType discord.ComponentType `json:"component_type"`
	CustomID      discord.ComponentID   `json:"custom_id"`
	// Values are the selected values of a select menu. For user, role,
	// mentionable and channel selects, these are the selected IDs.
	Values []string `json:"values,omitempty"`
}

// ModalInteractionData is the data of a modal that a user submits.
//
// https://docs.discord.food/interactions/receiving-and-responding#sendable-modal-submit-data-structure
type ModalInteractionData struct {
	// ID is the ID of the interaction that created the modal, from
	// gateway.InteractionModalCreateEvent.
	ID         discord.InteractionID  `json:"id"`
	CustomID   discord.ComponentID    `json:"custom_id"`
	Components []ModalSubmitComponent `json:"components"`
}

// ModalSubmitComponent is the value of a component in a submitted modal. It
// mirrors the modal's layout: labels have a Component, and action rows have
// Components.
type ModalSubmitComponent struct {
	Type     discord.ComponentType `json:"type"`
	CustomID discord.ComponentID   `json:"custom_id,omitempty"`
	// Value is a string for text inputs and radio groups, or a bool for
	// checkboxes.
	Value any `json:"value,omitempty"`
	// Values are the selected values of select menus and checkbox groups.
	Values []string `json:"values,omitempty"`

	Component  *ModalSubmitComponent  `json:"component,omitempty"`
	Components []ModalSubmitComponent `json:"components,omitempty"`
}

// SendInteraction sends an interaction as a user. Discord replies through the
// gateway with an InteractionSuccessEvent or InteractionFailureEvent, plus an
// InteractionModalCreateEvent if the application responds with a modal.
//
// This endpoint is undocumented and only works for user accounts.
//
// https://docs.discord.food/interactions/receiving-and-responding#create-interaction
func (c *Client) SendInteraction(data SendInteractionData) error {
	return c.FastRequest("POST", Endpoint+"interactions", httputil.WithJSONBody(data))
}
