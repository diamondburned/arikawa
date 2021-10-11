package discord

import (
	"errors"
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json"
)

// ErrNestedActionRow is returned if an action row is nested inside another
// action row.
var ErrNestedActionRow = errors.New("action row cannot have action row as a child")

// ComponentError describes an error in a component within the component tree.
type ComponentError struct {
	Component Component
	Errors    []httputil.HTTPError
}

// Error returns the first error's message or nil if none.
func (err ComponentError) Error() string {
	if len(err.Errors) == 0 {
		return ""
	}
	return `"` + err.Errors[0].Message + `"`
}

type componentError struct {
	Components map[int]componentError
	Errors     []httputil.HTTPError `json:"_errors"`
}

// WhereComponentError converts a 50035 component error to a list of Component
// errors from the given list of nested components. It should only be used for
// debugging. A nil slice is returned if the error isn't a component error.
func WhereComponentError(err error, components []Component) []ComponentError {
	var httpError *httputil.HTTPError
	if !errors.As(err, &httpError) || httpError.Code != 50035 {
		return nil
	}

	var errorBody struct {
		Errors struct {
			Data componentError
		}
	}

	if err := json.Unmarshal(httpError.Body, &errorBody); err != nil {
		return nil
	}

	return traverseComponentError(errorBody.Errors.Data, components)
}

func traverseComponentError(err componentError, cs []Component) []ComponentError {
	var errs []ComponentError

	for ix, err := range err.Components {
		if ix < 0 || ix >= len(cs) {
			continue
		}

		if len(err.Errors) > 0 {
			errs = append(errs, ComponentError{
				Component: cs[ix],
				Errors:    err.Errors,
			})
		}

		if len(err.Components) > 0 {
			if row, ok := cs[ix].(*ActionRowComponent); ok {
				errs = append(errs, traverseComponentError(err, *row)...)
			}
		}
	}

	return errs
}

// ComponentType is the type of a component.
type ComponentType uint

const (
	_ ComponentType = iota
	ActionRowComponentType
	ButtonComponentType
	SelectComponentType
)

// String formats Type's name as a string.
func (t ComponentType) String() string {
	switch t {
	case ActionRowComponentType:
		return "ActionRow"
	case ButtonComponentType:
		return "Button"
	case SelectComponentType:
		return "Select"
	default:
		return fmt.Sprintf("Type(%d)", int(t))
	}
}

type boxedComponent struct {
	Component
}

// UnmarshalJSON unmarshals JSON into the component.
func (c *boxedComponent) UnmarshalJSON(b []byte) error {
	comp, err := ParseComponent(b)
	if err != nil {
		return err
	}

	c.Component = comp
	return nil
}

// Component is a component that can be attached to an interaction response. To
// use Component for unmarshaling JSON, use the discord.Component type.
//
// The possible types are:
//
//    - ActionRow
//    - Button
//    - Select
//    - Unknown
//
type Component interface {
	json.Marshaler
	Type() ComponentType
}

// NewComponent returns a new Component from the given type that's matched with
// the global ComponentFunc map. If the type is unknown, then Unknown is used.
func ParseComponent(b []byte) (Component, error) {
	var t struct {
		Type ComponentType
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}

	var c Component
	var err error

	switch t.Type {
	case ActionRowComponentType:
		v := ActionRowComponent{}
		err = json.Unmarshal(b, &v)
		c = v
	case ButtonComponentType:
		v := ButtonComponent{}
		err = json.Unmarshal(b, &v)
		c = v
	case SelectComponentType:
		v := SelectComponent{}
		err = json.Unmarshal(b, &v)
		c = v
	default:
		v := UnknownComponent{typ: t.Type}
		err = json.Unmarshal(b, &v)
		c = v
	}

	return c, err
}

// ActionRow is a row of components at the bottom of a message.
type ActionRowComponent []Component // `json:"components"`

// Components wraps the given list of components inside ActionRows if it's not
// already in one. This is a convenient function that wraps components inside
// ActionRows for the user. It panincs if any of the action rows have nested
// action rows in them.
//
// Here's an example of how to use it:
//
//    discord.Components(
//        discord.TextButtonComponent("Hello, world!"),
//        discord.Components(
//            discord.TextButtonComponent("Hello!"),
//            discord.TextButtonComponent("Delete."),
//        ),
//    )
//
func Components(components ...Component) []Component {
	new := make([]Component, len(components))

	for i, comp := range components {
		ar, ok := comp.(ActionRowComponent)
		if !ok {
			comp = ActionRowComponent{comp}
		}

		for j, comp := range ar {
			if comp.Type() == ActionRowComponentType {
				log.Panicf("given components[%d][%d] is nested action row", i, j)
			}
		}

		new[i] = comp
	}

	return new
}

// ActionRowComponents creates a new action row component consisting of multiple
// components. If any of the components inside are of type ActionRowComponent,
// then the function panics.
func ActionRowComponents(components ...Component) Component {
	for i, component := range components {
		if component.Type() == ActionRowComponentType {
			log.Panicf("given components[%d] is nested action row", i)
		}
	}

	return ActionRowComponent(components)
}

// Type implements the Component interface.
func (a ActionRowComponent) Type() ComponentType {
	return ActionRowComponentType
}

// MarshalJSON marshals the action row in the format Discord expects.
func (a ActionRowComponent) MarshalJSON() ([]byte, error) {
	var actionRow struct {
		Type       ComponentType `json:"type"`
		Components []Component   `json:"components"`
	}

	actionRow.Components = a
	actionRow.Type = a.Type()

	return json.Marshal(actionRow)
}

// UnmarshalJSON unmarshals json into the components.
func (a *ActionRowComponent) UnmarshalJSON(b []byte) error {
	var rowTypes struct {
		Components []boxedComponent `json:"components"`
	}

	if err := json.Unmarshal(b, &rowTypes); err != nil {
		return err
	}

	*a = nil
	if len(rowTypes.Components) == 0 {
		return nil
	}

	*a = make([]Component, len(rowTypes.Components))
	for i, comp := range rowTypes.Components {
		(*a)[i] = comp.Component
	}

	return nil
}

// CustomID is the type for a component's custom ID.
type ComponentID string

// ComponentEmoji is the emoji displayed on the button before the text. For more
// information, see Emoji.
type ComponentEmoji struct {
	ID       EmojiID `json:"id,omitempty"`
	Name     string  `json:"name,omitempty"`
	Animated bool    `json:"animated,omitempty"`
}

// Button is a clickable button that may be added to an interaction
// response.
type ButtonComponent struct {
	// Style is one of the button styles.
	Style ButtonComponentStyle `json:"style"`
	// CustomID attached to InteractionCreate event when clicked.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Label is the text that appears on the button. It can have maximum 100
	// characters.
	Label string `json:"label,omitempty"`
	// Emoji should have Name, ID and Animated filled.
	Emoji *ComponentEmoji `json:"emoji,omitempty"`
	// Disabled determines whether the button is disabled.
	Disabled bool `json:"disabled,omitempty"`
}

// TextButtonComponent creates a new button with the given label used for the label and
// the custom ID.
func TextButtonComponent(style ButtonComponentStyle, label string) Component {
	return ButtonComponent{
		Style:    style,
		Label:    label,
		CustomID: ComponentID(label),
	}
}

// Type implements the Component interface.
func (b ButtonComponent) Type() ComponentType {
	return ButtonComponentType
}

// ButtonComponentStyle is the style to display a button in. Use one of the
// ButtonStyle constructor functions.
type ButtonComponentStyle interface {
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
func PrimaryButtonStyle() ButtonComponentStyle { return primaryButtonStyle }

// SecondaryButtonStyle is a style for a grey button.
func SecondaryButtonStyle() ButtonComponentStyle { return secondaryButtonStyle }

// SuccessButtonStyle is a style for a green button.
func SuccessButtonStyle() ButtonComponentStyle { return successButtonStyle }

// DangerButtonStyle is a style for a red button.
func DangerButtonStyle() ButtonComponentStyle { return dangerButtonStyle }

type linkButtonStyle URL

func (s linkButtonStyle) style() int { return int(linkButtonStyleNum) }

// LinkButtonStyle is a button style that navigates to a URL.
func LinkButtonStyle(url URL) ButtonComponentStyle { return linkButtonStyle(url) }

// MarshalJSON marshals the button in the format Discord expects.
func (b ButtonComponent) MarshalJSON() ([]byte, error) {
	type button ButtonComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		button
		URL URL `json:"url,omitempty"`
	}

	msg := Msg{
		Type:   ButtonComponentType,
		button: button(b),
	}

	if b.Style == nil {
		b.Style = PrimaryButtonStyle() // Sane default for button.
	}

	if link, ok := b.Style.(linkButtonStyle); ok {
		msg.URL = URL(link)
	}

	return json.Marshal(msg)
}

// Select is a clickable button that may be added to an interaction
// response.
type SelectComponent struct {
	// Options are the choices in the select.
	Options []SelectOption `json:"options"`
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
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
func (s SelectComponent) Type() ComponentType {
	return SelectComponentType
}

// MarshalJSON marshals the select in the format Discord expects.
func (s SelectComponent) MarshalJSON() ([]byte, error) {
	type sel SelectComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		sel
		MinValues *int `json:"min_values,omitempty"`
		MaxValues *int `json:"max_values,omitempty"`
	}

	msg := Msg{
		Type: SelectComponentType,
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
type UnknownComponent struct {
	json.Raw
	typ ComponentType
}

// Type implements the Component interface.
func (u UnknownComponent) Type() ComponentType {
	return u.typ
}

// ComponentResponseData is a union component interaction response types. The
// types can be whatever the constructors for this type will return. Underlying
// types of Response are all value types.
type ComponentResponseData interface {
	Type() ComponentType
	resp()
}

// SelectComponentResponse is a select component's response.
type SelectComponentResponse struct {
	CustomID ComponentID
	Values   []string
}

// NewSelectComponentResponse creates a new select component response.
func NewSelectComponentResponse(id ComponentID, values []string) ComponentResponseData {
	return SelectComponentResponse{
		CustomID: id,
		Values:   values,
	}
}

// Type implements Response.
func (r SelectComponentResponse) Type() ComponentType { return SelectComponentType }
func (r SelectComponentResponse) resp()               {}

// ButtonComponentResponse is a button component's response. It is the custom ID of the
// button within the component tree.
type ButtonComponentResponse struct {
	CustomID ComponentID
}

// NewButtonComponentResponse creates a new button component response.
func NewButtonComponentResponse(id ComponentID) ComponentResponseData {
	return ButtonComponentResponse{id}
}

// Type implements Response.
func (r ButtonComponentResponse) Type() ComponentType { return ButtonComponentType }
func (r ButtonComponentResponse) resp()               {}

// ParseComponentResponse parses the given bytes as a component response.
func ParseComponentResponse(b []byte) (ComponentResponseData, error) {
	var t struct {
		Type     ComponentType
		CustomID ComponentID `json:"custom_id"`
		Values   []string    `json:"values"`
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}

	var r ComponentResponseData
	var err error

	switch t.Type {
	case ButtonComponentType:
		v := ButtonComponentResponse{
			CustomID: t.CustomID,
		}
		err = json.Unmarshal(b, &v)
		r = v
	case SelectComponentType:
		v := SelectComponentResponse{
			CustomID: t.CustomID,
			Values:   t.Values,
		}
		err = json.Unmarshal(b, &v)
		r = v
	default:
		return nil, fmt.Errorf("unknown component response type %s", t)
	}

	return r, err
}
