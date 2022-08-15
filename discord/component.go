package discord

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/v3/internal/rfutil"
	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/pkg/errors"
)

// ComponentType is the type of a component.
type ComponentType uint

const (
	_ ComponentType = iota
	ActionRowComponentType
	ButtonComponentType
	SelectComponentType
	TextInputComponentType
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
	case TextInputComponentType:
		return "TextInput"
	default:
		return fmt.Sprintf("ComponentType(%d)", int(t))
	}
}

// ContainerComponents is primarily used for unmarshaling. It is the top-level
// type for component lists.
type ContainerComponents []ContainerComponent

// Find finds any component with the given custom ID.
func (c *ContainerComponents) Find(customID ComponentID) Component {
	for _, component := range *c {
		switch component := component.(type) {
		case *ActionRowComponent:
			if component := component.Find(customID); component != nil {
				return component
			}
		}
	}
	return nil
}

// Unmarshal unmarshals the components into the struct pointer v. Each struct
// field must be exported and is of a supported type.
//
// Fields that don't satisfy any of the above are ignored. The "discord" struct
// tag with a value "-" is ignored. Fields that aren't found in the list of
// options and have a "?" at the end of the "discord" struct tag are ignored.
//
// Each struct field will be used to search the tree of components for a
// matching custom ID. The struct must be a flat struct that lists all the
// components it needs using the custom ID.
//
// Supported Types
//
// The following types are supported:
//
//    - string (SelectComponent if range = [n, 1], TextInputComponent)
//    - bool (ButtonComponent or any component, true if present)
//    - []string (SelectComponent)
//
// Any types that are derived from any of the above built-in types are also
// supported.
//
// Pointer types to any of the above types are also supported and will also
// implicitly imply optionality.
func (c *ContainerComponents) Unmarshal(v interface{}) error {
	rv, rt, err := rfutil.StructValue(v)
	if err != nil {
		return err
	}

	numField := rt.NumField()
	for i := 0; i < numField; i++ {
		fieldStruct := rt.Field(i)
		if !fieldStruct.IsExported() {
			continue
		}

		name := fieldStruct.Tag.Get("discord")
		switch name {
		case "-":
			continue
		case "?":
			name = fieldStruct.Name + "?"
		case "":
			name = fieldStruct.Name
		}

		component := c.Find(ComponentID(strings.TrimSuffix(name, "?")))
		fieldv := rv.Field(i)
		fieldt := fieldStruct.Type

		if strings.HasSuffix(name, "?") {
			name = strings.TrimSuffix(name, "?")
			if component == nil {
				// not found
				continue
			}
		} else if fieldStruct.Type.Kind() == reflect.Ptr {
			fieldt = fieldt.Elem()
			if component == nil {
				// not found
				fieldv.Set(reflect.NewAt(fieldt, nil))
				continue
			}
			// found, so allocate new value and use that to set
			newv := reflect.New(fieldt)
			fieldv.Set(newv)
			fieldv = newv.Elem()
		} else if component == nil {
			// not found AND the field is not a pointer, so error out
			return fmt.Errorf("component %q is required but not found", name)
		}

		switch fieldt.Kind() {
		case reflect.Bool:
			// Intended for ButtonComponents.
			fieldv.Set(reflect.ValueOf(true).Convert(fieldt))
		case reflect.String:
			var v string

			switch component := component.(type) {
			case *TextInputComponent:
				v = component.Value.Val
			case *SelectComponent:
				switch len(component.Options) {
				case 0:
					// ok
				case 1:
					v = component.Options[0].Value
				default:
					return fmt.Errorf("component %q selected more than one item (bug, check ValueRange)", name)
				}
			default:
				return fmt.Errorf("component %q is of unsupported type %T", name, component)
			}

			fieldv.Set(reflect.ValueOf(v).Convert(fieldt))
		case reflect.Slice:
			elemt := fieldt.Elem()

			switch elemt.Kind() {
			case reflect.String:
				switch component := component.(type) {
				case *SelectComponent:
					fieldv.Set(reflect.MakeSlice(fieldt, len(component.Options), len(component.Options)))
					for i, option := range component.Options {
						fieldv.Index(i).Set(reflect.ValueOf(option.Value).Convert(elemt))
					}
				default:
					return fmt.Errorf("component %q is of unsupported type %T", name, component)
				}
			default:
				return fmt.Errorf("field %s (%q) has unknown slice type %s", fieldStruct.Name, name, fieldt)
			}
		default:
			return fmt.Errorf("field %s (%q) has unknown type %s", fieldStruct.Name, name, fieldt)
		}
	}

	return nil
}

// UnmarshalJSON unmarshals JSON into the component. It does type-checking and
// will only accept container components.
func (c *ContainerComponents) UnmarshalJSON(b []byte) error {
	var jsons []json.Raw
	if err := json.Unmarshal(b, &jsons); err != nil {
		return err
	}

	*c = make([]ContainerComponent, len(jsons))

	for i, b := range jsons {
		p, err := ParseComponent(b)
		if err != nil {
			return err
		}

		cc, ok := p.(ContainerComponent)
		if !ok {
			return fmt.Errorf("expected container, got %T", p)
		}
		(*c)[i] = cc
	}

	return nil
}

// Component is a component that can be attached to an interaction response. A
// Component is either an InteractiveComponent or a ContainerComponent. See
// those appropriate types for more information.
//
// The following types satisfy this interface:
//
//    - *ActionRowComponent
//    - *ButtonComponent
//    - *SelectComponent
//    - *TextInputComponent
//
type Component interface {
	// Type returns the type of the underlying component.
	Type() ComponentType
	_cmp()
}

// InteractiveComponent extends the Component for components that are
// interactible, or components that aren't containers (like ActionRow). This is
// useful for ActionRow to type-check that no nested ActionRows are allowed.
//
// The following types satisfy this interface:
//
//    - *ButtonComponent
//    - *SelectComponent
//    - *TextInputComponent
//
type InteractiveComponent interface {
	Component
	// ID returns the ID of the underlying component.
	ID() ComponentID
	_icp()
}

// ContainerComponent is the opposite of InteractiveComponent: it describes
// components that only contain other components. The only component that
// satisfies that is ActionRow.
//
// The following types satisfy this interface:
//
//    - *ActionRowComponent
//
type ContainerComponent interface {
	Component
	_ctn()
}

// NewComponent returns a new Component from the given type that's matched with
// the global ComponentFunc map. If the type is unknown, then Unknown is used.
func ParseComponent(b []byte) (Component, error) {
	var t struct {
		Type ComponentType
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal component type")
	}

	var c Component

	switch t.Type {
	case ActionRowComponentType:
		c = &ActionRowComponent{}
	case ButtonComponentType:
		c = &ButtonComponent{}
	case SelectComponentType:
		c = &SelectComponent{}
	case TextInputComponentType:
		c = &TextInputComponent{}
	default:
		c = &UnknownComponent{typ: t.Type}
	}

	if err := json.Unmarshal(b, c); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal component body")
	}

	return c, nil
}

// ActionRow is a row of components at the bottom of a message. Its type,
// InteractiveComponent, ensures that only non-ActionRow components are allowed
// on it.
type ActionRowComponent []InteractiveComponent

// Components wraps the given list of components inside ActionRows if it's not
// already in one. This is a convenient function that wraps components inside
// ActionRows for the user. It panics if any of the action rows have nested
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
func Components(components ...Component) ContainerComponents {
	new := make([]ContainerComponent, len(components))

	for i, comp := range components {
		cc, ok := comp.(ContainerComponent)
		if !ok {
			// Wrap. We're asserting that comp is either a ContainerComponent or
			// an InteractiveComponent. Neither would be a bug, therefore panic.
			cc = &ActionRowComponent{comp.(InteractiveComponent)}
		}

		new[i] = cc
	}

	return new
}

// ComponentsPtr returns the pointer to Components' return. This is a
// convenient function.
func ComponentsPtr(components ...Component) *ContainerComponents {
	v := Components(components...)
	return &v
}

// Type implements the Component interface.
func (a *ActionRowComponent) Type() ComponentType {
	return ActionRowComponentType
}

func (a *ActionRowComponent) _cmp() {}
func (a *ActionRowComponent) _ctn() {}

// Find finds any component with the given custom ID.
func (a *ActionRowComponent) Find(customID ComponentID) Component {
	for _, component := range *a {
		if component.ID() == customID {
			return component
		}
	}
	return nil
}

// MarshalJSON marshals the action row in the format Discord expects.
func (a *ActionRowComponent) MarshalJSON() ([]byte, error) {
	var actionRow struct {
		Type       ComponentType           `json:"type"`
		Components *[]InteractiveComponent `json:"components"`
	}

	actionRow.Components = (*[]InteractiveComponent)(a)
	actionRow.Type = a.Type()

	return json.Marshal(actionRow)
}

// UnmarshalJSON unmarshals JSON into the components. It does type-checking and
// will only accept interactive components.
func (a *ActionRowComponent) UnmarshalJSON(b []byte) error {
	var row struct {
		Components []json.Raw `json:"components"`
	}

	if err := json.Unmarshal(b, &row); err != nil {
		return err
	}

	*a = make(ActionRowComponent, len(row.Components))

	for i, b := range row.Components {
		p, err := ParseComponent(b)
		if err != nil {
			return errors.Wrapf(err, "failed to parse component %d", i)
		}

		ic, ok := p.(InteractiveComponent)
		if !ok {
			return fmt.Errorf("expected interactive, got %T", p)
		}
		(*a)[i] = ic
	}

	return nil
}

// ComponentID is the type for a component's custom ID. It is NOT a snowflake,
// but rather a user-defined opaque string.
type ComponentID string

// ComponentEmoji is the emoji displayed on the button before the text. For more
// information, see Emoji.
type ComponentEmoji struct {
	ID       EmojiID `json:"id,omitempty"`
	Name     string  `json:"name,omitempty"`
	Animated bool    `json:"animated,omitempty"`
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
	basicButtonStyleLen
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
func TextButtonComponent(style ButtonComponentStyle, label string) ButtonComponent {
	return ButtonComponent{
		Style:    style,
		Label:    label,
		CustomID: ComponentID(label),
	}
}

// ID implements the Component interface.
func (b *ButtonComponent) ID() ComponentID { return b.CustomID }

// Type implements the Component interface.
func (b *ButtonComponent) Type() ComponentType {
	return ButtonComponentType
}

func (b *ButtonComponent) _cmp() {}
func (b *ButtonComponent) _icp() {}

// MarshalJSON marshals the button in the format Discord expects.
func (b *ButtonComponent) MarshalJSON() ([]byte, error) {
	if b.Style == nil {
		b.Style = PrimaryButtonStyle() // Sane default for button.
	}

	type button ButtonComponent

	type Msg struct {
		*button
		Type  ComponentType `json:"type"`
		Style int           `json:"style"`
		URL   URL           `json:"url,omitempty"`
	}

	msg := Msg{
		Type:   ButtonComponentType,
		Style:  b.Style.style(),
		button: (*button)(b),
	}

	if link, ok := b.Style.(linkButtonStyle); ok {
		msg.URL = URL(link)
	}

	return json.Marshal(msg)
}

// UnmarshalJSON unmarshals a component JSON into the button. It does NOT do
// type-checking; use ParseComponent for that.
func (b *ButtonComponent) UnmarshalJSON(j []byte) error {
	type button ButtonComponent

	msg := struct {
		*button
		Style basicButtonStyle `json:"style"`
		URL   URL              `json:"url,omitempty"`
	}{
		button: (*button)(b),
	}

	if err := json.Unmarshal(j, &msg); err != nil {
		return err
	}

	if 0 > msg.Style || msg.Style >= basicButtonStyleLen {
		return fmt.Errorf("unknown button style %d", msg.Style)
	}

	switch msg.Style {
	case linkButtonStyleNum:
		b.Style = LinkButtonStyle(msg.URL)
	default:
		b.Style = msg.Style
	}

	return nil
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
	Emoji *ComponentEmoji `json:"emoji,omitempty"`
	// Default will render this option as selected by default if true.
	Default bool `json:"default,omitempty"`
}

// ID implements the Component interface.
func (s *SelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *SelectComponent) Type() ComponentType {
	return SelectComponentType
}

func (s *SelectComponent) _cmp() {}
func (s *SelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *SelectComponent) MarshalJSON() ([]byte, error) {
	type sel SelectComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues *int `json:"min_values,omitempty"`
		MaxValues *int `json:"max_values,omitempty"`
	}

	msg := Msg{
		Type: SelectComponentType,
		sel:  (*sel)(s),
	}

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
}

type TextInputStyle uint8

const (
	_ TextInputStyle = iota
	TextInputShortStyle
	TextInputParagraphStyle
)

// TextInputComponents provide a user-facing text box to be filled out. They can only
// be used with modals.
type TextInputComponent struct {
	// CustomID provides a developer-defined ID for the input (max 100 chars)
	CustomID ComponentID `json:"custom_id"`
	// Style determines if the component should use the short or paragraph style
	Style TextInputStyle `json:"style"`
	// Label is the title of this component, describing its use
	Label string `json:"label"`
	// LengthLimits is the minimum and maximum length for the input
	LengthLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Value is the pre-filled value of this component (max 4000 chars)
	Value option.NullableString `json:"value,omitempty"`
	// Placeholder is the text that appears when the input is empty (max 100 chars)
	Placeholder option.NullableString `json:"placeholder,omitempty"`
}

func (s *TextInputComponent) _cmp() {}
func (s *TextInputComponent) _icp() {}

func (i *TextInputComponent) ID() ComponentID {
	return i.CustomID
}

func (i *TextInputComponent) Type() ComponentType {
	return TextInputComponentType
}

func (i *TextInputComponent) MarshalJSON() ([]byte, error) {
	type text TextInputComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*text
		MinLength *int `json:"min_length,omitempty"`
		MaxLength *int `json:"max_length,omitempty"`
	}

	m := Msg{
		Type: i.Type(),
		text: (*text)(i),
	}

	if i.LengthLimits != [2]int{0, 0} {
		m.MinLength = new(int)
		m.MaxLength = new(int)

		*m.MinLength = i.LengthLimits[0]
		*m.MaxLength = i.LengthLimits[1]
	}
	return json.Marshal(m)
}

// Unknown is reserved for components with unknown or not yet implemented
// components types. It can also be used in place of a ComponentInteraction.
type UnknownComponent struct {
	json.Raw
	id  ComponentID
	typ ComponentType
}

// ID implements the Component and ComponentInteraction interfaces.
func (u *UnknownComponent) ID() ComponentID { return u.id }

// Type implements the Component and ComponentInteraction interfaces.
func (u *UnknownComponent) Type() ComponentType { return u.typ }

// Type implements InteractionData.
func (u *UnknownComponent) InteractionType() InteractionDataType {
	return ComponentInteractionType
}

func (u *UnknownComponent) resp() {}
func (u *UnknownComponent) data() {}
func (u *UnknownComponent) _cmp() {}
func (u *UnknownComponent) _icp() {}
