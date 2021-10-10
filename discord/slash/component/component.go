// Package component contains widget-like types for Discord application
// commands.
package component

import (
	"errors"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json"
)

var ErrNestedActionRow = errors.New("action row cannot have action row as a child")

// Type is the type of a component.
type Type uint

const (
	_ Type = iota
	ActionRowType
	ButtonType
	SelectType
)

// Boxed boxes Component for the purpose of JSON unmarshalling. Type assertions
// should be made on Component to access the underlying data. The underlying
// types of the Component are pointer types.
//
// This type is mostly used as a workaround for the limitations within the JSON
// library, so it should only be used for that purpose.
type Boxed struct {
	Component
}

// UnBoxComponents returns a slice of the underlying component interfaces.
func UnboxComponents(wraps []Boxed) []Component {
	components := make([]Component, len(wraps))
	for i, w := range wraps {
		components[i] = w.Component
	}

	return components
}

// Component is a component that can be attached to an interaction response. To
// use Component for unmarshaling JSON, use the Boxed type.
type Component interface {
	json.Marshaler
	Type() Type
}

// ComponentFunc maps known component types to its constructor types. Any
// constructor not in the map will
var ComponentFunc = map[Type]func() Component{
	ActionRowType: func() Component { return &ActionRow{} },
	ButtonType:    func() Component { return &Button{} },
	SelectType:    func() Component { return &Select{} },
}

func newComponent(t Type) Component {
	fn, ok := ComponentFunc[t]
	if !ok {
		return &Unknown{typ: t}
	}

	return fn()
}

// UnmarshalJSON unmarshals json into the component.
func (box *Boxed) UnmarshalJSON(b []byte) error {
	var t struct {
		Type Type `json:"type"`
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}

	box.Component = newComponent(t.Type)
	return json.Unmarshal(b, box.Component)
}

// ActionRow is a row of components at the bottom of a message.
type ActionRow []Component // `json:"components"`

// Type implements the Component interface.
func (ActionRow) Type() Type {
	return ActionRowType
}

// MarshalJSON marshals the action row in the format Discord expects.
func (a ActionRow) MarshalJSON() ([]byte, error) {
	var actionRow struct {
		Type       Type        `json:"type"`
		Components []Component `json:"components"`
	}

	actionRow.Components = a
	actionRow.Type = a.Type()

	return json.Marshal(actionRow)
}

// UnmarshalJSON unmarshals json into the components.
func (a *ActionRow) UnmarshalJSON(b []byte) error {
	var rowTypes struct {
		Components []struct {
			Type Type `json:"type"`
		} `json:"components"`
	}

	if err := json.Unmarshal(b, &rowTypes); err != nil {
		return err
	}

	components := make([]Component, len(rowTypes.Components))

	for i, t := range rowTypes.Components {
		if t.Type == ActionRowType {
			return ErrNestedActionRow
		}

		components[i] = newComponent(t.Type)
	}

	if err := json.Unmarshal(b, components); err != nil {
		return err
	}

	*a = ActionRow(components)
	return nil
}

// CustomID is the type for a component's custom ID.
type CustomID string

// Validate briefly checks that id is valid.
func (id CustomID) Validate() error {
	if len(id) > 100 {
		return fmt.Errorf("id too long (%d), max 100", len(id))
	}
	return nil
}

// Emoji is the emoji displayed on the button before the text. For more
// information, see discord.Emoji.
type Emoji struct {
	ID       discord.EmojiID `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Animated bool            `json:"animated,omitempty"`
}

// Button is a clickable button that may be added to an interaction
// response.
type Button struct {
	// Style is one of the button styles.
	Style ButtonStyle `json:"style"`
	// ID attached to InteractionCreate event when clicked.
	ID CustomID `json:"custom_id,omitempty"`
	// Label is the text that appears on the button. It can have maximum 100
	// characters.
	Label string `json:"label,omitempty"`
	// Emoji should have Name, ID and Animated filled.
	Emoji *Emoji `json:"emoji,omitempty"`
	// Disabled determines whether the button is disabled.
	Disabled bool `json:"disabled,omitempty"`
}

// Type implements the Component interface.
func (*Button) Type() Type {
	return ButtonType
}

// ButtonStyle is the style to display a button in. Use one of the ButtonStyle
// constructor functions.
type ButtonStyle interface {
	style() int
}

type basicButtonStyle int

func (s basicButtonStyle) style() int { return int(s) }

const (
	_ basicButtonStyle = iota
	primaryButtonStyle
	secondaryButtonStyle
	successButtonStyle
	dangerButtonStyle
	linkButtonStyleNum
)

// PrimaryButtonStyle is a style for a blurple button.
func PrimaryButtonStyle() ButtonStyle { return primaryButtonStyle }

// SecondaryButtonStyle is a style for a grey button.
func SecondaryButtonStyle() ButtonStyle { return secondaryButtonStyle }

// SuccessButtonStyle is a style for a green button.
func SuccessButtonStyle() ButtonStyle { return successButtonStyle }

// DangerButtonStyle is a style for a red button.
func DangerButtonStyle() ButtonStyle { return dangerButtonStyle }

type linkButtonStyle discord.URL

func (s linkButtonStyle) style() int { return int(linkButtonStyleNum) }

// LinkButtonStyle is a button style that navigates to a URL.
func LinkButtonStyle(url discord.URL) ButtonStyle { return linkButtonStyle(url) }

// MarshalJSON marshals the button in the format Discord expects.
func (b Button) MarshalJSON() ([]byte, error) {
	type button Button

	type Msg struct {
		Type Type `json:"type"`
		button
		URL discord.URL `json:"url,omitempty"`
	}

	msg := Msg{
		Type:   ButtonType,
		button: button(b),
	}

	if b.Style == nil {
		b.Style = PrimaryButtonStyle() // Sane default for button.
	}

	if link, ok := b.Style.(linkButtonStyle); ok {
		msg.URL = discord.URL(link)
	}

	return json.Marshal(msg)
}

// Select is a clickable button that may be added to an interaction
// response.
type Select struct {
	// Options are the choices in the select.
	Options []SelectOption `json:"options"`
	// ID is the custom unique ID.
	ID CustomID `json:"custom_id,omitempty"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Disabled disables the select if true.
	Disabled bool `json:"disabled,omitempty"`
}

// SelectOption is an option in the select component.
type SelectOption struct {
	// Label is the user-facing name of the option. Max 100 characters.
	Label string `json:"label"`
	// Value is the internal value that is echoed back to the program. It's
	// similar to the custom ID. Max 100 characters.
	Value string `json:"value"`
	// Description is the additional description of an option.
	Description string `json:"description,omitempty"`
	// Emoji is the optional emoji object.
	Emoji *Emoji `json:"emoji,omitempty"`
	// Default will render this option as selected by default if true.
	Default bool `json:"default,omitempty"`
}

// Type implements the Component interface.
func (Select) Type() Type {
	return SelectType
}

// MarshalJSON marshals the select in the format Discord expects.
func (s Select) MarshalJSON() ([]byte, error) {
	type sel Select

	type Msg struct {
		Type Type `json:"type"`
		sel
		MinValues *int `json:"min_values,omitempty"`
		MaxValues *int `json:"max_values,omitempty"`
	}

	msg := Msg{
		Type: SelectType,
		sel:  sel(s),
	}

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
}

// Unknown is reserved for components with unknown or not yet implemented
// components types.
type Unknown struct {
	json.Raw
	typ Type
}

// Type implements the Component interface.
func (u *Unknown) Type() Type {
	return u.typ
}
