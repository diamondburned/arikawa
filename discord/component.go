package discord

import "encoding/json"

// ComponentType is the type of a component.
type ComponentType uint

const (
	ActionRowComponent ComponentType = iota + 1
	ButtonComponent
)

// Component is a component that can be attached to an interaction response.
type Component interface {
	json.Marshaler
	Type() ComponentType
}

// ActionRow is a row of components at the bottom of a message.
type ActionRow struct {
	Components []Component `json:"components"`
}

// Type implements the InteractionComponent interface.
func (ActionRow) Type() ComponentType {
	return ActionRowComponent
}

// MarshalJSON marshals the action row in the format Discord expects.
func (a ActionRow) MarshalJSON() ([]byte, error) {
	type actionRow ActionRow

	return json.Marshal(struct {
		actionRow
		Type ComponentType `json:"type"`
	}{
		actionRow: actionRow(a),
		Type:      ActionRowComponent,
	})
}

// Button is a clickable button that may be added to an interaction response.
type Button struct {
	Label string `json:"label"`
	// CustomID attached to InteractionCreate event when clicked.
	CustomID string       `json:"custom_id"`
	Style    ButtonStyle  `json:"style"`
	Emoji    *ButtonEmoji `json:"emoji,omitempty"`
	// Present on link-style buttons.
	URL      string `json:"url,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
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

// Type implements the InteractionComponent interface.
func (Button) Type() ComponentType {
	return ButtonComponent
}

// MarshalJSON marshals the button in the format Discord expects.
func (b Button) MarshalJSON() ([]byte, error) {
	type button Button

	if b.Style == 0 {
		b.Style = PrimaryButton // Sane default for button.
	}

	return json.Marshal(struct {
		button
		Type ComponentType `json:"type"`
	}{
		button: button(b),
		Type:   ButtonComponent,
	})
}
