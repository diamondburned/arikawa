package discord

import (
	"strings"

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

	// Locale is the selected language of the invoking user. It is returned in
	// all interactions except ping interactions. Use this Locale field to
	// obtain the language of the user who used the interaction.
	Locale Language `json:"locale,omitempty"`
	// GuildLocale is the guild's preferred locale, if invoked in a guild.
	GuildLocale string `json:"guild_locale,omitempty"`
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
		return nil // Ping isn't actually an object.
	case CommandInteractionType:
		e.Data = &CommandInteraction{}
	case ComponentInteractionType:
		d, err := ParseComponentInteraction(target.Data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal component interaction event data")
		}
		e.Data = d
		return nil
	case AutocompleteInteractionType:
		e.Data = &AutocompleteInteraction{}
	case ModalInteractionType:
		e.Data = &ModalInteraction{}
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
	if e.Data.InteractionType() == 0 {
		return nil, errors.New("unexpected 0 InteractionEvent.Data.Type")
	}

	v := struct {
		Type InteractionDataType `json:"type"`
		*event
	}{
		Type:  e.Data.InteractionType(),
		event: (*event)(e),
	}

	return json.Marshal(v)
}

// InteractionDataType is the type of each Interaction, enumerated in
// integers.
type InteractionDataType uint

const (
	PingInteractionType InteractionDataType = iota + 1
	CommandInteractionType
	ComponentInteractionType
	AutocompleteInteractionType
	ModalInteractionType
)

// InteractionData holds the respose data of an interaction, or more
// specifically, the data that Discord sends to us. Type assertions should be
// made on it to access the underlying data.
//
// The following types implement this interface:
//
//    - *PingInteraction
//    - *AutocompleteInteraction
//    - *CommandInteraction
//    - *SelectInteraction (also ComponentInteraction)
//    - *ButtonInteraction (also ComponentInteraction)
//
type InteractionData interface {
	InteractionType() InteractionDataType
	data()
}

// PingInteraction is a ping Interaction response.
type PingInteraction struct{}

// InteractionType implements InteractionData.
func (*PingInteraction) InteractionType() InteractionDataType { return PingInteractionType }
func (*PingInteraction) data()                                {}

// AutocompleteInteraction is an autocompletion Interaction response.
type AutocompleteInteraction struct {
	CommandID CommandID `json:"id"`

	// Name of command autocomplete is triggered for.
	Name        string              `json:"name"`
	CommandType CommandType         `json:"type"`
	Version     string              `json:"version"`
	Options     AutocompleteOptions `json:"options"`
}

// Type implements ComponentInteraction.
func (*AutocompleteInteraction) InteractionType() InteractionDataType {
	return AutocompleteInteractionType
}
func (*AutocompleteInteraction) data() {}

// AutocompleteOptions is a list of autocompletion options.
// Use `Find` to get your named autocompletion option.
type AutocompleteOptions []AutocompleteOption

// Find returns the named autocomplete option.
func (o AutocompleteOptions) Find(name string) AutocompleteOption {
	for _, opt := range o {
		if strings.EqualFold(opt.Name, name) {
			return opt
		}
	}
	return AutocompleteOption{}
}

// AutocompleteOption is an autocompletion option in an AutocompleteInteraction.
type AutocompleteOption struct {
	Type    CommandOptionType    `json:"type"`
	Name    string               `json:"name"`
	Value   json.Raw             `json:"value"`
	Focused bool                 `json:"focused"`
	Options []AutocompleteOption `json:"options"`
}

// String will return the value if the option's value is a valid string.
// Otherwise, it will return the raw JSON value of the other type.
func (o AutocompleteOption) String() string {
	var value string
	if err := json.Unmarshal(o.Value, &value); err != nil {
		return string(o.Value)
	}
	return value
}

// IntValue reads the option's value as an int.
func (o AutocompleteOption) IntValue() (int64, error) {
	var i int64
	err := o.Value.UnmarshalTo(&i)
	return i, err
}

// BoolValue reads the option's value as a bool.
func (o AutocompleteOption) BoolValue() (bool, error) {
	var b bool
	err := o.Value.UnmarshalTo(&b)
	return b, err
}

// SnowflakeValue reads the option's value as a snowflake.
func (o AutocompleteOption) SnowflakeValue() (Snowflake, error) {
	var id Snowflake
	err := o.Value.UnmarshalTo(&id)
	return id, err
}

// FloatValue reads the option's value as a float64.
func (o AutocompleteOption) FloatValue() (float64, error) {
	var f float64
	err := o.Value.UnmarshalTo(&f)
	return f, err
}

// ComponentInteraction is a union component interaction response types. The
// types can be whatever the constructors for this type will return.
//
// The following types implement this interface:
//
//    - *SelectInteraction
//    - *ButtonInteraction
//
type ComponentInteraction interface {
	InteractionData
	// ID returns the ID of the component in response. Not all component
	// interactions will have a component ID.
	ID() ComponentID
	// Type returns the type of the component in response.
	Type() ComponentType
	resp()
}

// SelectInteraction is a select component's response.
type SelectInteraction struct {
	CustomID ComponentID `json:"custom_id"`
	Values   []string    `json:"values"`
}

// ID implements ComponentInteraction.
func (s *SelectInteraction) ID() ComponentID { return s.CustomID }

// Type implements ComponentInteraction.
func (s *SelectInteraction) Type() ComponentType { return SelectComponentType }

// InteractionType implements InteractionData.
func (s *SelectInteraction) InteractionType() InteractionDataType {
	return ComponentInteractionType
}

func (s *SelectInteraction) resp() {}
func (s *SelectInteraction) data() {}

// ButtonInteraction is a button component's response. It is the custom ID of
// the button within the component tree.
type ButtonInteraction struct {
	CustomID ComponentID `json:"custom_id"`
}

// ID implements ComponentInteraction.
func (b *ButtonInteraction) ID() ComponentID { return b.CustomID }

// Type implements ComponentInteraction.
func (b *ButtonInteraction) Type() ComponentType { return ButtonComponentType }

// InteractionType implements InteractionData.
func (b *ButtonInteraction) InteractionType() InteractionDataType {
	return ComponentInteractionType
}

func (b *ButtonInteraction) data() {}
func (b *ButtonInteraction) resp() {}

// ParseComponentInteraction parses the given bytes as a component response.
func ParseComponentInteraction(b []byte) (ComponentInteraction, error) {
	var t struct {
		Type     ComponentType `json:"component_type"`
		CustomID ComponentID   `json:"custom_id"`
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal component interaction header")
	}

	var d ComponentInteraction

	switch t.Type {
	case ButtonComponentType:
		d = &ButtonInteraction{CustomID: t.CustomID}
	case SelectComponentType:
		d = &SelectInteraction{CustomID: t.CustomID}
	default:
		d = &UnknownComponent{
			Raw: append(json.Raw(nil), b...),
			id:  t.CustomID,
			typ: t.Type,
		}
		return d, nil
	}

	if err := json.Unmarshal(b, d); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal component interaction data")
	}

	return d, nil
}

// CommandInteractionOptions is a list of interaction options.
// Use `Find` to get your named interaction option
type CommandInteractionOptions []CommandInteractionOption

// CommandInteraction is an application command interaction that Discord sends
// to us.
type CommandInteraction struct {
	ID      CommandID                 `json:"id"`
	Name    string                    `json:"name"`
	Options CommandInteractionOptions `json:"options"`
	// 	GuildID is the id of the guild the command is registered to
	GuildID GuildID `json:"guild_id,omitempty"`
	// TargetID is the id of the user or message targeted by a user or message command.
	//
	// See TargetUserID and TargetMessageID
	TargetID Snowflake `json:"target_id,omitempty"`
	Resolved struct {
		// User contains user objects.
		Users map[UserID]User `json:"users,omitempty"`
		// Members contains partial member objects (missing User, Deaf and
		// Mute).
		Members map[UserID]Member `json:"members,omitempty"`
		// Role contains role objects.
		Roles map[RoleID]Role `json:"roles,omitempty"`
		// Channels contains partial channel objects that only have ID, Name,
		// Type and Permissions. Threads will also have ThreadMetadata and
		// ParentID.
		Channels map[ChannelID]Channel `json:"channels,omitempty"`
		// Messages contains partial message objects. All fields without
		// omitempty are presumably present.
		Messages map[MessageID]Message `json:"messages,omitempty"`
		// Attachments contains attachments objects.
		Attachments map[AttachmentID]Attachment `json:"attachments,omitempty"`
	}
}

// InteractionType implements InteractionData.
func (*CommandInteraction) InteractionType() InteractionDataType {
	return CommandInteractionType
}

// TargetUserID is the id of the user targeted by a user command
func (c *CommandInteraction) TargetUserID() UserID {
	return UserID(c.TargetID)
}

// TargetMessageID is the id of the message targeted by a message command
func (c *CommandInteraction) TargetMessageID() MessageID {
	return MessageID(c.TargetID)
}

func (*CommandInteraction) data() {}

// CommandInteractionOption is an option for a Command interaction response.
type CommandInteractionOption struct {
	Type    CommandOptionType         `json:"type"`
	Name    string                    `json:"name"`
	Value   json.Raw                  `json:"value"`
	Options CommandInteractionOptions `json:"options"`
}

// Find returns the named command option
func (o CommandInteractionOptions) Find(name string) CommandInteractionOption {
	for _, opt := range o {
		if strings.EqualFold(opt.Name, name) {
			return opt
		}
	}
	return CommandInteractionOption{}
}

// String will return the value if the option's value is a valid string.
// Otherwise, it will return the raw JSON value of the other type.
func (o CommandInteractionOption) String() string {
	var value string
	if err := json.Unmarshal(o.Value, &value); err != nil {
		return string(o.Value)
	}
	return value
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

// ModalInteraction is the submitted modal form
type ModalInteraction struct {
	CustomID   ComponentID         `json:"custom_id"`
	Components ContainerComponents `json:"components"`
}

// InteractionType implements InteractionData.
func (m *ModalInteraction) InteractionType() InteractionDataType {
	return ModalInteractionType
}

func (m *ModalInteraction) data() {}

// UnknownInteractionData describes an Interaction response with an unknown
// type.
type UnknownInteractionData struct {
	json.Raw
	typ InteractionDataType
}

// InteractionType implements InteractionData.
func (u *UnknownInteractionData) InteractionType() InteractionDataType {
	return u.typ
}

func (u *UnknownInteractionData) data() {}
