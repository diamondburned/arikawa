package discord

import (
	"errors"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var ErrNestedActionRow = errors.New("action row cannot have action row as a child")

// ComponentType is the type of a component.
type ComponentType uint

const (
	ActionRowComponentType ComponentType = iota + 1
	ButtonComponentType
	SelectComponentType
)

// ComponentWrap wraps Component for the purpose of JSON unmarshalling.
// Type assertions should be made on Component to access the underlying data.
// The underlying types of the Component are pointer types.
type ComponentWrap struct {
	Component Component
}

// UnwrapComponents returns a slice of the underlying component interfaces.
func UnwrapComponents(wraps []ComponentWrap) []Component {
	components := make([]Component, len(wraps))
	for i, w := range wraps {
		components[i] = w.Component
	}

	return components
}

// Type returns the underlying component's type.
func (c *ComponentWrap) Type() ComponentType {
	return c.Component.Type()
}

// MarshalJSON marshals the component in the format Discord expects.
func (c *ComponentWrap) MarshalJSON() ([]byte, error) {
	return c.Component.MarshalJSON()
}

// UnmarshalJSON unmarshals json into the component.
func (c *ComponentWrap) UnmarshalJSON(b []byte) error {
	var t struct {
		Type ComponentType `json:"type"`
	}

	err := json.Unmarshal(b, &t)
	if err != nil {
		return err
	}

	switch t.Type {
	case ActionRowComponentType:
		c.Component = &ActionRowComponent{}
	case ButtonComponentType:
		c.Component = &ButtonComponent{}
	default:
		c.Component = &UnknownComponent{typ: t.Type}
	}

	return json.Unmarshal(b, c.Component)
}

// Component is a component that can be attached to an interaction response.
type Component interface {
	json.Marshaler
	Type() ComponentType
}

// ActionRowComponent is a row of components at the bottom of a message.
type ActionRowComponent struct {
	Components []Component `json:"components"`
}

// Type implements the Component interface.
func (*ActionRowComponent) Type() ComponentType {
	return ActionRowComponentType
}

// MarshalJSON marshals the action row in the format Discord expects.
func (a ActionRowComponent) MarshalJSON() ([]byte, error) {
	type actionRow ActionRowComponent

	return json.Marshal(struct {
		actionRow
		Type ComponentType `json:"type"`
	}{
		actionRow: actionRow(a),
		Type:      ActionRowComponentType,
	})
}

// UnmarshalJSON unmarshals json into the components.
func (a *ActionRowComponent) UnmarshalJSON(b []byte) error {
	type actionRow ActionRowComponent

	type rowTypes struct {
		Components []struct {
			Type ComponentType `json:"type"`
		} `json:"components"`
	}

	var r rowTypes
	err := json.Unmarshal(b, &r)
	if err != nil {
		return err
	}

	a.Components = make([]Component, len(r.Components))
	for i, t := range r.Components {
		switch t.Type {
		case ActionRowComponentType:
			// ActionRow cannot have child components of type Actionrow
			return ErrNestedActionRow
		case ButtonComponentType:
			a.Components[i] = &ButtonComponent{}
		default:
			a.Components[i] = &UnknownComponent{typ: t.Type}
		}
	}

	alias := actionRow(*a)
	err = json.Unmarshal(b, &alias)
	if err != nil {
		return err
	}

	*a = ActionRowComponent(alias)

	return nil
}

// ButtonComponent is a clickable button that may be added to an interaction
// response.
type ButtonComponent struct {
	Label string `json:"label"`
	// CustomID attached to InteractionCreate event when clicked.
	CustomID string       `json:"custom_id"`
	Style    ButtonStyle  `json:"style"`
	Emoji    *ButtonEmoji `json:"emoji,omitempty"`
	// URL is only present on link-style buttons.
	URL      URL  `json:"url,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
}

// Type implements the Component interface.
func (*ButtonComponent) Type() ComponentType {
	return ButtonComponentType
}

// ButtonStyle is the style to display a button in.
type ButtonStyle uint

// All types of ButtonStyle documented.
const (
	// PrimaryButton is a blurple button.
	PrimaryButton ButtonStyle = iota + 1
	// SecondaryButton is a grey button.
	SecondaryButton
	// SuccessButton is a green button.
	SuccessButton
	// DangerButton is a red button.
	DangerButton
	// LinkButton is a button that navigates to a URL.
	LinkButton
)

// ButtonEmoji is the emoji displayed on the button before the text.
type ButtonEmoji struct {
	Name     string  `json:"name,omitempty"`
	ID       EmojiID `json:"id,omitempty"`
	Animated bool    `json:"animated,omitempty"`
}

// MarshalJSON marshals the button in the format Discord expects.
func (b ButtonComponent) MarshalJSON() ([]byte, error) {
	type button ButtonComponent

	if b.Style == 0 {
		b.Style = PrimaryButton // Sane default for button.
	}

	return json.Marshal(struct {
		button
		Type ComponentType `json:"type"`
	}{
		button: button(b),
		Type:   ButtonComponentType,
	})
}

// SelectComponent is a clickable button that may be added to an interaction
// response.
type SelectComponent struct {
	CustomID    string                  `json:"custom_id"`
	Options     []SelectComponentOption `json:"options"`
	Placeholder string                  `json:"placeholder,omitempty"`
	MinValues   option.Int              `json:"min_values,omitempty"`
	MaxValues   int                     `json:"max_values,omitempty"`
	Disabled    bool                    `json:"disabled,omitempty"`
}

type SelectComponentOption struct {
	Label       string       `json:"label"`
	Value       string       `json:"value"`
	Description string       `json:"description,omitempty"`
	Emoji       *ButtonEmoji `json:"emoji,omitempty"`
	Default     bool         `json:"default,omitempty"`
}

// Type implements the Component interface.
func (*SelectComponent) Type() ComponentType {
	return SelectComponentType
}

// MarshalJSON marshals the select in the format Discord expects.
func (s SelectComponent) MarshalJSON() ([]byte, error) {
	type selectComponent SelectComponent

	return json.Marshal(struct {
		selectComponent
		Type ComponentType `json:"type"`
	}{
		selectComponent: selectComponent(s),
		Type:            SelectComponentType,
	})
}

// UnknownComponent is reserved for components with unknown or not yet
// implemented components types.
type UnknownComponent struct {
	json.Raw
	typ ComponentType
}

// Type implements the Component interface.
func (u *UnknownComponent) Type() ComponentType {
	return u.typ
}
