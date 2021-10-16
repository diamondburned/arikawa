package discord

import (
	"strconv"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

// InteractionEvent describes the full incoming interaction event. It may be a
// gateway event or a webhook event.
//
// https://discord.com/developers/docs/topics/gateway#interactions
type InteractionEvent struct {
	ID        InteractionID       `json:"id"`
	AppID     AppID               `json:"application_id"`
	Data      InteractionResponse `json:"data"`
	ChannelID ChannelID           `json:"channel_id,omitempty"`
	Token     string              `json:"token"`
	Version   int                 `json:"version"`

	// Message is the message the component was attached to.
	// Only present for component interactions, not command interactions.
	Message *Message `json:"message,omitempty"`

	// Member is only present if this came from a guild. To get a user, use the
	// Sender method.
	Member  *Member `json:"member,omitempty"`
	GuildID GuildID `json:"guild_id,omitempty"`

	// User is only present if this didn't come from a guild. To get a user, use
	// the Sender method.
	User *User `json:"user,omitempty"`
}

// Sender returns the sender of this event from either the Member field or the
// User field. If neither of those fields are available, then nil is returned.
func (e *InteractionEvent) Sender() *User {
	if e.User != nil {
		return e.User
	}
	if e.Member != nil {
		return &e.Member.User
	}
	return nil
}

// SenderID returns the sender's ID. See Sender for more information. If Sender
// returns nil, then 0 is returned.
func (e *InteractionEvent) SenderID() UserID {
	if sender := e.Sender(); sender != nil {
		return sender.ID
	}
	return 0
}

func (e *InteractionEvent) UnmarshalJSON(b []byte) error {
	type event InteractionEvent

	v := struct {
		Type InteractionResponseType `json:"type"`
		Data json.Raw                `json:"data"`
		*event
	}{
		event: (*event)(e),
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	var err error

	switch v.Type {
	case PingResponseType:
		e.Data = PingResponse{}
	case ComponentResponseType:
		v := ComponentResponse{}
		err = json.Unmarshal(b, &v)
		e.Data = v
	case CommandResponseType:
		v := CommandResponse{}
		err = json.Unmarshal(b, &v)
		e.Data = v
	default:
		v := UnknownInteractionResponse{typ: v.Type}
		err = json.Unmarshal(b, &v)
		e.Data = v
	}

	return err
}

func (e *InteractionEvent) MarshalJSON() ([]byte, error) {
	type event InteractionEvent

	if e.Data == nil {
		return nil, errors.New("missing InteractionEvent.Data")
	}
	if e.Data.Type() == 0 {
		return nil, errors.New("unexpected 0 InteractionEvent.Data.Type")
	}

	v := struct {
		Type InteractionResponseType `json:"type"`
		*event
	}{
		Type:  e.Data.Type(),
		event: (*event)(e),
	}

	return json.Marshal(v)
}

// InteractionResponseType is the type of each Interaction, enumerated in
// integers.
type InteractionResponseType uint

const (
	_ InteractionResponseType = iota
	PingResponseType
	CommandResponseType
	ComponentResponseType
)

// InteractionResponse holds the respose data of an interaction. Type assertions should be
// made on it to access the underlying data. The underlying types of the
// Responses are value types.
type InteractionResponse interface {
	Type() InteractionResponseType
	data()
}

// PingResponse is a ping Interaction response.
type PingResponse struct{}

// NewPingResponse creates a new Ping response.
func NewPingResponse() InteractionResponse {
	return PingResponse{}
}

// Type implements Response.
func (PingResponse) Type() InteractionResponseType { return PingResponseType }
func (PingResponse) data()                         {}

// ComponentResponse is a component Interaction response.
type ComponentResponse struct {
	ComponentResponseData
}

// NewComponentResponse creates a new Component response.
func NewComponentResponse(data ComponentResponseData) InteractionResponse {
	return ComponentResponse{
		ComponentResponseData: data,
	}
}

// Type implements Response.
func (ComponentResponse) Type() InteractionResponseType { return ComponentResponseType }
func (ComponentResponse) data()                         {}

func (r *ComponentResponse) UnmarshalJSON(b []byte) error {
	resp, err := ParseComponentResponse(b)
	if err != nil {
		return err
	}

	r.ComponentResponseData = resp
	return nil
}

func (r ComponentResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.ComponentResponseData)
}

// CommandResponse is a command response.
type CommandResponse struct {
	ID      CommandID      `json:"id"`
	Name    string         `json:"name"`
	Options CommandOptions `json:"options"`
}

// NewCommandResponse creates a new Command interaction.
func NewCommandResponse(data *CommandResponse) InteractionResponse {
	return data
}

// Type implements Response.
func (CommandResponse) Type() InteractionResponseType { return CommandResponseType }
func (CommandResponse) data()                         {}

// CommandResponseOption is an option for a Command interaction response.
type CommandResponseOption struct {
	Name    string                  `json:"name"`
	Value   json.Raw                `json:"value"`
	Options []CommandResponseOption `json:"options"`
}

// String will return the value if the option's value is a valid string.
// Otherwise, it will return the raw JSON value of the other type.
func (o CommandResponseOption) String() string {
	val := string(o.Value)

	s, err := strconv.Unquote(val)
	if err != nil {
		return val
	}

	return s
}

// IntValue reads the option's value as an int.
func (o CommandResponseOption) IntValue() (int64, error) {
	var i int64
	err := o.Value.UnmarshalTo(&i)
	return i, err
}

// BoolValue reads the option's value as a bool.
func (o CommandResponseOption) BoolValue() (bool, error) {
	var b bool
	err := o.Value.UnmarshalTo(&b)
	return b, err
}

// SnowflakeValue reads the option's value as a snowflake.
func (o CommandResponseOption) SnowflakeValue() (Snowflake, error) {
	var id Snowflake
	err := o.Value.UnmarshalTo(&id)
	return id, err
}

// FloatValue reads the option's value as a float64.
func (o CommandResponseOption) FloatValue() (float64, error) {
	var f float64
	err := o.Value.UnmarshalTo(&f)
	return f, err
}

// UnknownInteractionResponse describes an Interaction response with an unknown
// type.
type UnknownInteractionResponse struct {
	json.Raw
	typ InteractionResponseType
}

// Type implements Interaction.
func (u UnknownInteractionResponse) Type() InteractionResponseType { return u.typ }
func (u UnknownInteractionResponse) data()                         {}
