package discord

import "github.com/diamondburned/arikawa/v2/utils/json"

// ComponentType is the type of a component.
type ComponentType uint

const (
	ActionRowComponentType ComponentType = iota + 1
	ButtonComponentType
)

// Component is a component that can be attached to an interaction response.
type Component struct {
	// Data is an interface that contains a type of component such as Button or
	// ActionRow.
	Data interface {
		json.Marshaler
		Type() ComponentType
	}
}

// Type returns the component's type.
func (c Component) Type() ComponentType {
	return c.Data.Type()
}

// MarshalJSON marshals the component in the format Discord expects.
func (c *Component) MarshalJSON() ([]byte, error) {
	return c.Data.MarshalJSON()
}

// UnmarshalJSON unmarshals json into the component.
func (c *Component) UnmarshalJSON(b []byte) error {
	var t struct {
		Type ComponentType `json:"type"`
	}

	err := json.Unmarshal(b, &t)
	if err != nil {
		return err
	}

	switch t.Type {
	case ActionRowComponentType:
		c.Data = &ActionRowComponent{}
	case ButtonComponentType:
		c.Data = &ButtonComponent{}
	default:
		c.Data = &UnknownComponent{typ: t.Type}
	}

	return json.Unmarshal(b, c.Data)
}

// ActionRow is a row of components at the bottom of a message.
type ActionRowComponent struct {
	Components []Component `json:"components"`
}

// Type implements the Component Data interface.
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

// Type implements the Component Data interface.
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

// Type implements the Component Data interface.
func (u UnknownComponent) Type() ComponentType {
	return u.typ
}
