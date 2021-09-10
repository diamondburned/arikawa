package discord

import (
	"strconv"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

// https://discord.com/developers/docs/topics/gateway#interactions
type Interaction struct {
	ID        InteractionID   `json:"id"`
	AppID     AppID           `json:"application_id"`
	Type      InteractionType `json:"type"`
	Data      InteractionData `json:"data,omitempty"`
	ChannelID ChannelID       `json:"channel_id,omitempty"`
	Token     string          `json:"token"`
	Version   int             `json:"version"`

	// Message is the message the component was attached to.
	// Only present for component interactions, not command interactions.
	Message *Message `json:"message,omitempty"`

	// Member is only present if this came from a guild.
	Member  *Member `json:"member,omitempty"`
	GuildID GuildID `json:"guild_id,omitempty"`

	// User is only present if this didn't come from a guild.
	User *User `json:"user,omitempty"`
}

func (i *Interaction) UnmarshalJSON(p []byte) error {
	type interaction Interaction
	v := struct {
		Data json.Raw `json:"data,omitempty"`
		*interaction
	}{interaction: (*interaction)(i)}
	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}

	switch v.Type {
	case PingInteraction:
		return nil
	case ComponentInteraction:
		i.Data = &ComponentInteractionData{}
	case CommandInteraction:
		i.Data = &CommandInteractionData{}
	default:
		i.Data = &UnknownInteractionData{typ: v.Type}
	}

	return json.Unmarshal(v.Data, i.Data)
}

type InteractionType uint

const (
	PingInteraction InteractionType = iota + 1
	CommandInteraction
	ComponentInteraction
)

// InteractionData holds the data of an interaction.
// Type assertions should be made on InteractionData to access the underlying data.
// The underlying types of the InteractionData are pointer types.
type InteractionData interface {
	Type() InteractionType
}

type ComponentInteractionData struct {
	CustomID      string        `json:"custom_id"`
	ComponentType ComponentType `json:"component_type"`
	Values        []string      `json:"values"`
}

func (*ComponentInteractionData) Type() InteractionType {
	return ComponentInteraction
}

type CommandInteractionData struct {
	ID      CommandID           `json:"id"`
	Name    string              `json:"name"`
	Options []InteractionOption `json:"options"`
}

func (*CommandInteractionData) Type() InteractionType {
	return CommandInteraction
}

type UnknownInteractionData struct {
	json.Raw
	typ InteractionType
}

func (u *UnknownInteractionData) Type() InteractionType {
	return u.typ
}

type InteractionOption struct {
	Name    string              `json:"name"`
	Value   json.Raw            `json:"value"`
	Options []InteractionOption `json:"options"`
}

// String will return the value if the option's value is a valid string.
// Otherwise, it will return the raw JSON value of the other type.
func (o InteractionOption) String() string {
	val := string(o.Value)
	s, err := strconv.Unquote(val)
	if err != nil {
		return val
	}
	return s
}

func (o InteractionOption) Int() (int64, error) {
	var i int64
	err := o.Value.UnmarshalTo(&i)
	return i, err
}

func (o InteractionOption) Bool() (bool, error) {
	var b bool
	err := o.Value.UnmarshalTo(&b)
	return b, err
}

func (o InteractionOption) Snowflake() (Snowflake, error) {
	var id Snowflake
	err := o.Value.UnmarshalTo(&id)
	return id, err
}

func (o InteractionOption) Float() (float64, error) {
	var f float64
	err := o.Value.UnmarshalTo(&f)
	return f, err
}
