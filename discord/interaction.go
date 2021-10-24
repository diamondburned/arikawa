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
	ID        InteractionID   `json:"id"`
	Data      InteractionData `json:"data"`
	AppID     AppID           `json:"application_id"`
	ChannelID ChannelID       `json:"channel_id,omitempty"`
	Token     string          `json:"token"`
	Version   int             `json:"version"`

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

	target := struct {
		Type InteractionDataType `json:"type"`
		Data json.Raw            `json:"data"`
		*event
	}{
		event: (*event)(e),
	}

	if err := json.Unmarshal(b, &target); err != nil {
		return err
	}

	var err error

	switch target.Type {
	case PingInteractionType:
		e.Data = &PingInteraction{}
	case ComponentInteractionType:
		e.Data = &ComponentInteraction{}
	case CommandInteractionType:
		e.Data = &CommandInteraction{}
	default:
		e.Data = &UnknownInteractionData{
			Raw: target.Data,
			typ: target.Type,
		}
		return nil
	}

	if err := json.Unmarshal(target.Data, e.Data); err != nil {
		return errors.Wrap(err, "failed to unmarshal interaction event data")
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
		Type InteractionDataType `json:"type"`
		*event
	}{
		Type:  e.Data.Type(),
		event: (*event)(e),
	}

	return json.Marshal(v)
}

// InteractionDataType is the type of each Interaction, enumerated in
// integers.
type InteractionDataType uint

const (
	_ InteractionDataType = iota
	PingInteractionType
	CommandInteractionType
	ComponentInteractionType
)

// InteractionData holds the respose data of an interaction, or more
// specifically, the data that Discord sends to us. Type assertions should be
// made on it to access the underlying data. The underlying types of the
// Responses are value types. See the constructors for the possible types.
type InteractionData interface {
	Type() InteractionDataType
	data()
}

// PingInteraction is a ping Interaction response.
type PingInteraction struct{}

// Type implements Response.
func (*PingInteraction) Type() InteractionDataType { return PingInteractionType }
func (*PingInteraction) data()                     {}

// ComponentInteraction is a component Interaction response.
type ComponentInteraction struct {
	ComponentInteractionData
}

// Type implements Response.
func (*ComponentInteraction) Type() InteractionDataType { return ComponentInteractionType }
func (*ComponentInteraction) data()                     {}

func (r *ComponentInteraction) UnmarshalJSON(b []byte) error {
	resp, err := ParseComponentInteraction(b)
	if err != nil {
		return err
	}

	r.ComponentInteractionData = resp
	return nil
}

func (r *ComponentInteraction) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.ComponentInteractionData)
}

// CommandInteraction is a command interaction that Discord sends to us.
type CommandInteraction struct {
	ID      CommandID      `json:"id"`
	Name    string         `json:"name"`
	Options CommandOptions `json:"options"`
}

// Type implements Response.
func (*CommandInteraction) Type() InteractionDataType { return CommandInteractionType }
func (*CommandInteraction) data()                     {}

// CommandInteractionOption is an option for a Command interaction response.
type CommandInteractionOption struct {
	Name    string                     `json:"name"`
	Value   json.Raw                   `json:"value"`
	Options []CommandInteractionOption `json:"options"`
}

// String will return the value if the option's value is a valid string.
// Otherwise, it will return the raw JSON value of the other type.
func (o CommandInteractionOption) String() string {
	val := string(o.Value)

	s, err := strconv.Unquote(val)
	if err != nil {
		return val
	}

	return s
}

// IntValue reads the option's value as an int.
func (o CommandInteractionOption) IntValue() (int64, error) {
	var i int64
	err := o.Value.UnmarshalTo(&i)
	return i, err
}

// BoolValue reads the option's value as a bool.
func (o CommandInteractionOption) BoolValue() (bool, error) {
	var b bool
	err := o.Value.UnmarshalTo(&b)
	return b, err
}

// SnowflakeValue reads the option's value as a snowflake.
func (o CommandInteractionOption) SnowflakeValue() (Snowflake, error) {
	var id Snowflake
	err := o.Value.UnmarshalTo(&id)
	return id, err
}

// FloatValue reads the option's value as a float64.
func (o CommandInteractionOption) FloatValue() (float64, error) {
	var f float64
	err := o.Value.UnmarshalTo(&f)
	return f, err
}

// UnknownInteractionData describes an Interaction response with an unknown
// type.
type UnknownInteractionData struct {
	json.Raw
	typ InteractionDataType
}

// Type implements Interaction.
func (u *UnknownInteractionData) Type() InteractionDataType { return u.typ }
func (u *UnknownInteractionData) data()                     {}
