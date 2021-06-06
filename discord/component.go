package discord

import (
	"errors"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

var ErrNestedActionRow = errors.New("action row cannot have action row as a child")

// ComponentType is the type of a component.
type ComponentType uint

const (
	ActionRowComponentType ComponentType = iota + 1
	ButtonComponentType
)

// ComponentWrap wraps Component for the purpose of JSON unmarshalling.
// Type assetions should be made on Component to access the underlying data.
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
func (c ComponentWrap) Type() ComponentType {
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

// ActionRow is a row of components at the bottom of a message.
type ActionRowComponent struct {
	Components []Component `json:"components"`
}

// Type implements the Component interface.
func (ActionRowComponent) Type() ComponentType {
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

// Button is a clickable button that may be added to an interaction response.
type ButtonComponent struct {
	Label string `json:"label"`
	// CustomID attached to InteractionCreate event when clicked.
	CustomID string       `json:"custom_id"`
	Style    ButtonStyle  `json:"style"`
	Emoji    *ButtonEmoji `json:"emoji,omitempty"`
	// Present on link-style buttons.
	URL      string `json:"url,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Type implements the Component interface.
func (ButtonComponent) Type() ComponentType {
	return ButtonComponentType
}

// ButtonStyle is the style to display a button in.
type ButtonStyle uint

// All types of ButtonStyle documented.
const (
	PrimaryButton   ButtonStyle = iota + 1 // Blurple button.
	SecondaryButton                        // Grey button.
	SuccessButton                          // Green button.
	DangerButton                           // Red button.
	LinkButton                             // Button that navigates to a URL.
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

// UnknownComponent is reserved for components with unknown or not yet
// implemented components types.
type UnknownComponent struct {
	json.Raw
	typ ComponentType
}

// Type implements the Component interface.
func (u UnknownComponent) Type() ComponentType {
	return u.typ
}
