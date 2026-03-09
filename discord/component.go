package discord

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/internal/rfutil"
	"github.com/diamondburned/arikawa/v3/utils/json"
)

// ComponentType is the type of a component.
type ComponentType uint

const (
	ActionRowComponentType ComponentType = iota + 1
	ButtonComponentType
	StringSelectComponentType
	TextInputComponentType
	UserSelectComponentType
	RoleSelectComponentType
	MentionableSelectComponentType
	ChannelSelectComponentType
	SectionComponentType
	TextDisplayComponentType
	ThumbnailComponentType
	MediaGalleryComponentType
	FileComponentType
	SeparatorComponentType
	_
	ContentInventoryEntryType
	ContainerComponentType
	LabelComponentType
	FileUploadComponentType
	_
	RadioGroupComponentType
	CheckboxGroupComponentType
	CheckboxComponentType
)

// String formats Type's name as a string.
func (t ComponentType) String() string {
	switch t {
	case ActionRowComponentType:
		return "ActionRow"
	case ButtonComponentType:
		return "Button"
	case StringSelectComponentType:
		return "StringSelect"
	case TextInputComponentType:
		return "TextInput"
	case UserSelectComponentType:
		return "User"
	case RoleSelectComponentType:
		return "Role"
	case MentionableSelectComponentType:
		return "Mentionable"
	case ChannelSelectComponentType:
		return "Channel"
	case SectionComponentType:
		return "Section"
	case TextDisplayComponentType:
		return "TextDisplay"
	case ThumbnailComponentType:
		return "Thumbnail"
	case MediaGalleryComponentType:
		return "MediaGallery"
	case FileComponentType:
		return "File"
	case SeparatorComponentType:
		return "Separator"
	case ContainerComponentType:
		return "Container"
	case LabelComponentType:
		return "Label"
	case FileUploadComponentType:
		return "FileUpload"
	case RadioGroupComponentType:
		return "RadioGroup"
	case CheckboxGroupComponentType:
		return "CheckboxGroup"
	case CheckboxComponentType:
		return "Checkbox"
	default:
		return fmt.Sprintf("ComponentType(%d)", int(t))
	}
}

// TopLevelComponents is primarily used for unmarshaling. It is the top-level
// type for component lists.
type TopLevelComponents []TopLevelComponent

// Find finds any component with the given custom ID.
func (c *TopLevelComponents) Find(customID ComponentID) Component {
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
// # Supported Types
//
// The following types are supported:
//
//   - string (SelectComponent if range = [n, 1], TextInputComponent)
//   - int*, uint*, float* (uses Parse{Int,Uint,Float}, SelectComponent if range = [n, 1], TextInputComponent)
//   - bool (ButtonComponent or any component, true if present)
//   - []string (SelectComponent)
//
// Any types that are derived from any of the above built-in types are also
// supported.
//
// Pointer types to any of the above types are also supported and will also
// implicitly imply optionality.
func (c *TopLevelComponents) Unmarshal(v any) error {
	rv, rt, err := rfutil.StructValue(v)
	if err != nil {
		return err
	}

	numField := rt.NumField()
	for i := range numField {
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

		if before, ok := strings.CutSuffix(name, "?"); ok {
			name = before
			if component == nil {
				// not found
				continue
			}
		} else if fieldStruct.Type.Kind() == reflect.Pointer {
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

		switch fieldk := fieldt.Kind(); fieldk {
		case reflect.Bool:
			// Intended for ButtonComponents.
			fieldv.Set(reflect.ValueOf(true).Convert(fieldt))
		case reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			var v string

			switch component := component.(type) {
			case *TextInputComponent:
				v = component.Value
			case *StringSelectComponent:
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

			switch fieldk {
			case reflect.String:
				fieldv.Set(reflect.ValueOf(v).Convert(fieldt))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i, err := strconv.ParseInt(v, 10, rfutil.KindBits(fieldk))
				if err != nil {
					return fmt.Errorf("component %q has invalid integer: %w", name, err)
				}
				fieldv.Set(reflect.ValueOf(i).Convert(fieldt))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				u, err := strconv.ParseUint(v, 10, rfutil.KindBits(fieldk))
				if err != nil {
					return fmt.Errorf("component %q has invalid unsigned (positive) integer: %w", name, err)
				}
				fieldv.Set(reflect.ValueOf(u).Convert(fieldt))
			case reflect.Float32, reflect.Float64:
				f, err := strconv.ParseFloat(v, rfutil.KindBits(fieldk))
				if err != nil {
					return fmt.Errorf("component %q has invalid floating-point number: %w", name, err)
				}
				fieldv.Set(reflect.ValueOf(f).Convert(fieldt))
			default:
				panic("unreachable")
			}
		case reflect.Slice:
			elemt := fieldt.Elem()

			switch elemt.Kind() {
			case reflect.String:
				switch component := component.(type) {
				case *StringSelectComponent:
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
func (c *TopLevelComponents) UnmarshalJSON(b []byte) error {
	var jsons []json.Raw
	if err := json.Unmarshal(b, &jsons); err != nil {
		return err
	}

	*c = make([]TopLevelComponent, len(jsons))

	for i, b := range jsons {
		p, err := ParseComponent(b)
		if err != nil {
			return err
		}

		cc, ok := p.(TopLevelComponent)
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
//   - *ActionRowComponent
//   - *ButtonComponent
//   - *StringSelectComponent
//   - *TextInputComponent
//   - *UserSelectComponent
//   - *RoleSelectComponent
//   - *MentionableSelectComponent
//   - *ChannelSelectComponent
//   - *SectionComponent
//   - *TextDisplayComponent
//   - *ThumbnailComponent
//   - *MediaGalleryComponent
//   - *FileComponent
//   - *SeparatorComponent
//   - *LabelComponent
//   - *FileUploadComponent
//   - *RadioGroupComponent
//   - *CheckboxGroupComponent
//   - *CheckboxComponent
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
//   - *ButtonComponent
//   - *TextInputComponent
//   - *StringSelectComponent
//   - *UserSelectComponent
//   - *RoleSelectComponent
//   - *MentionableSelectComponent
//   - *ChannelSelectComponent
type InteractiveComponent interface {
	Component
	// ID returns the ID of the underlying component.
	ID() ComponentID
	_icp()
}

// TopLevelComponent is the opposite of InteractiveComponent: it describes
// components that only contain other components. The only component that
// satisfies that is ActionRow.
//
// The following types satisfy this interface:
//
//   - *ActionRowComponent
//   - *SectionComponent
//   - *MediaGalleryComponent
//   - *FileComponent
//   - *SeparatorComponent
//   - *ContainerComponent
//   - *LabelComponent
//   - *FileUploadComponent
type TopLevelComponent interface {
	Component
	_tlc()
}

// NewComponent returns a new Component from the given type that's matched with
// the global ComponentFunc map. If the type is unknown, then Unknown is used.
func ParseComponent(b []byte) (Component, error) {
	var t struct {
		Type ComponentType
	}

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type: %w", err)
	}

	var c Component
	switch t.Type {
	case ActionRowComponentType:
		c = &ActionRowComponent{}
	case ButtonComponentType:
		c = &ButtonComponent{}
	case StringSelectComponentType:
		c = &StringSelectComponent{}
	case TextInputComponentType:
		c = &TextInputComponent{}
	case UserSelectComponentType:
		c = &UserSelectComponent{}
	case RoleSelectComponentType:
		c = &RoleSelectComponent{}
	case MentionableSelectComponentType:
		c = &MentionableSelectComponent{}
	case ChannelSelectComponentType:
		c = &ChannelSelectComponent{}
	case SectionComponentType:
		c = &SectionComponent{}
	case TextDisplayComponentType:
		c = &TextDisplayComponent{}
	case ThumbnailComponentType:
		c = &ThumbnailComponent{}
	case MediaGalleryComponentType:
		c = &MediaGalleryComponent{}
	case FileComponentType:
		c = &FileComponent{}
	case SeparatorComponentType:
		c = &SeparatorComponent{}
	// ContentInventory not included since not in spec
	case ContainerComponentType:
		c = &ContainerComponent{}
	case LabelComponentType:
		c = &LabelComponent{}
	case FileUploadComponentType:
		c = &FileUploadComponent{}
	case RadioGroupComponentType:
		c = &RadioGroupComponent{}
	case CheckboxGroupComponentType:
		c = &CheckboxGroupComponent{}
	case CheckboxComponentType:
		c = &CheckboxComponent{}
	default:
		c = &UnknownComponent{typ: t.Type}
	}

	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component body: %w", err)
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
//	discord.Components(
//	    discord.TextButtonComponent("Hello, world!"),
//	    discord.Components(
//	        discord.TextButtonComponent("Hello!"),
//	        discord.TextButtonComponent("Delete."),
//	    ),
//	)
func Components(components ...Component) TopLevelComponents {
	new := make([]TopLevelComponent, len(components))

	for i, comp := range components {
		cc, ok := comp.(TopLevelComponent)
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
func ComponentsPtr(components ...Component) *TopLevelComponents {
	v := Components(components...)
	return &v
}

// Type implements the Component interface.
func (a *ActionRowComponent) Type() ComponentType {
	return ActionRowComponentType
}

func (a *ActionRowComponent) _cmp() {}
func (a *ActionRowComponent) _tlc() {}

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
			return fmt.Errorf("failed to parse component %d: %w", i, err)
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
	primaryButtonStyle basicButtonStyle = iota + 1
	secondaryButtonStyle
	successButtonStyle
	dangerButtonStyle
	linkButtonStyleNum
	premiumButtonStyle
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

// PremiumButtonStyle is a style for purchasing an SKU
func PremiumButtonStyle() ButtonComponentStyle { return premiumButtonStyle }

type linkButtonStyle URL

func (s linkButtonStyle) style() int { return int(linkButtonStyleNum) }

// LinkButtonStyle is a button style that navigates to a URL.
func LinkButtonStyle(url URL) ButtonComponentStyle { return linkButtonStyle(url) }

// Button is a clickable button that may be added to an interaction
// response.
type ButtonComponent struct {
	// Style is one of the button styles.
	Style ButtonComponentStyle `json:"style"`
	// Label is the text that appears on the button. It can have maximum 100
	// characters.
	Label string `json:"label,omitempty"`
	// Emoji should have Name, ID and Animated filled.
	Emoji *ComponentEmoji `json:"emoji,omitempty"`
	// CustomID attached to InteractionCreate event when clicked.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// SKU ID for thing to be purchased
	SKUID SKUID `json:"sku_id,omitempty"`
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

// StringSelectComponent is a dropdown menu that may be added to an interaction
// response.
type StringSelectComponent struct {
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Options are the choices in the select.
	Options []SelectOption `json:"options"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Whether the string select is required to answer in a modal (defaults to true)
	// The required field is only available for String Selects in modals. It is ignored in messages.
	Required bool `json:"required,omitempty"`
	// Disabled disables the select if true.
	// Using disabled in a modal will result in an error. Modals can not currently have disabled components in them.
	Disabled bool `json:"disabled,omitempty"`
	// 	The text of the selected options
	Values []string `json:"values,omitempty"`
}

// SelectOption is an option in the select component.
type SelectOption struct {
	// Label is the user-facing name of the option. Max 100 characters.
	Label string `json:"label"`
	// Value is the internal value that is echoed back to the program. It's
	// similar to the custom ID. Max 100 characters.
	Value string `json:"value"`
	// Description is the additional description of an option. Max 100 characters.
	Description string `json:"description,omitempty"`
	// Emoji is the optional emoji object.
	Emoji *ComponentEmoji `json:"emoji,omitempty"`
	// Default will render this option as selected by default if true.
	Default bool `json:"default,omitempty"`
}

// ID implements the Component interface.
func (s *StringSelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *StringSelectComponent) Type() ComponentType {
	return StringSelectComponentType
}

func (s *StringSelectComponent) _cmp() {}
func (s *StringSelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *StringSelectComponent) MarshalJSON() ([]byte, error) {
	type sel StringSelectComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues *int `json:"min_values,omitempty"`
		MaxValues *int `json:"max_values,omitempty"`
	}

	msg := Msg{
		Type: StringSelectComponentType,
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
	TextInputShortStyle TextInputStyle = iota + 1
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
	// Deprecated: Deprecated in favor of 'label' and 'description' on the label component
	Label string `json:"label,omitempty"`
	// LengthLimits is the minimum and maximum length for the input
	LengthLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Value is the pre-filled value of this component (max 4000 chars)
	Value string `json:"value,omitempty"`
	// Placeholder is the text that appears when the input is empty (max 100 chars)
	Placeholder string `json:"placeholder,omitempty"`
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

type UserSelectComponent struct {
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// DefaultUsers is the slice of UserIDs that are marked as selected by default
	DefaultUsers []UserID `json:"-"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Disabled disables the select if true.
	Disabled bool `json:"disabled,omitempty"`
	// IDs of the selected users
	Values []UserID `json:"values,omitempty"`
}

// ID implements the Component interface.
func (s *UserSelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *UserSelectComponent) Type() ComponentType {
	return UserSelectComponentType
}

func (s *UserSelectComponent) _cmp() {}
func (s *UserSelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *UserSelectComponent) MarshalJSON() ([]byte, error) {
	type sel UserSelectComponent

	type DefaultValue struct {
		Id   UserID `json:"id"`
		Type string `json:"type"`
	}

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues     *int           `json:"min_values,omitempty"`
		MaxValues     *int           `json:"max_values,omitempty"`
		DefaultValues []DefaultValue `json:"default_values,omitempty"`
	}

	msg := Msg{
		Type: UserSelectComponentType,
		sel:  (*sel)(s),
	}

	var defaultValues []DefaultValue

	if len(s.DefaultUsers) > 0 {
		for _, userId := range s.DefaultUsers {
			defaultValues = append(defaultValues, DefaultValue{Id: userId, Type: "user"})
		}
	}

	msg.DefaultValues = defaultValues

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
}

type RoleSelectComponent struct {
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// DefaultRoles is the slice of RoleIDs that are marked as selected by default
	DefaultRoles []RoleID `json:"-"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Disabled disables the select if true.
	Disabled bool `json:"disabled,omitempty"`
	// IDs of the selected roles
	Values []RoleID `json:"values,omitempty"`
}

// ID implements the Component interface.
func (s *RoleSelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *RoleSelectComponent) Type() ComponentType {
	return RoleSelectComponentType
}

func (s *RoleSelectComponent) _cmp() {}
func (s *RoleSelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *RoleSelectComponent) MarshalJSON() ([]byte, error) {
	type sel RoleSelectComponent

	type DefaultValue struct {
		Id   RoleID `json:"id"`
		Type string `json:"type"`
	}

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues     *int           `json:"min_values,omitempty"`
		MaxValues     *int           `json:"max_values,omitempty"`
		DefaultValues []DefaultValue `json:"default_values,omitempty"`
	}

	msg := Msg{
		Type: RoleSelectComponentType,
		sel:  (*sel)(s),
	}

	var defaultValues []DefaultValue

	if len(s.DefaultRoles) > 0 {
		for _, roleId := range s.DefaultRoles {
			defaultValues = append(defaultValues, DefaultValue{Id: roleId, Type: "role"})
		}
	}

	msg.DefaultValues = defaultValues

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
}

// DefaultMention type is a Union type which packs both UserID and RoleID
type DefaultMention struct {
	userId UserID `json:"-"`
	roleId RoleID `json:"-"`
}

// DefaultUserMention creates a new DefaultMention type with only UserID
func DefaultUserMention(userId UserID) DefaultMention {
	return DefaultMention{userId: userId}
}

// DefaultRoleMention creates a new DefaultMention type with only RoleID
func DefaultRoleMention(roleId RoleID) DefaultMention {
	return DefaultMention{roleId: roleId}
}

type MentionableSelectComponent struct {
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// DefaultMentions is the slice of User / Role Mentions that are selected by default
	// Example:
	//     DefaultMentions: []DefaultMention{
	//         discord.DefaultUserMention(0382080830233),
	// 	       discord.DefaultRoleMention(4820380382080),
	//         ...
	//     }
	DefaultMentions []DefaultMention `json:"-"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Disabled disables the select if true.
	Disabled bool `json:"disabled,omitempty"`
	// IDs of the selected mentionables
	Values []Snowflake `json:"values,omitempty"`
}

// ID implements the Component interface.
func (s *MentionableSelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *MentionableSelectComponent) Type() ComponentType {
	return MentionableSelectComponentType
}

func (s *MentionableSelectComponent) _cmp() {}
func (s *MentionableSelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *MentionableSelectComponent) MarshalJSON() ([]byte, error) {
	type sel MentionableSelectComponent

	type DefaultValue struct {
		Id   Snowflake `json:"id"`
		Type string    `json:"type"`
	}

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues     *int           `json:"min_values,omitempty"`
		MaxValues     *int           `json:"max_values,omitempty"`
		DefaultValues []DefaultValue `json:"default_values,omitempty"`
	}

	msg := Msg{
		Type: MentionableSelectComponentType,
		sel:  (*sel)(s),
	}

	var defaultValues []DefaultValue

	if len(s.DefaultMentions) > 0 {
		for _, mention := range s.DefaultMentions {
			if mention.userId.IsValid() {
				defaultValues =
					append(defaultValues, DefaultValue{Id: Snowflake(mention.userId), Type: "user"})
			}
			if mention.roleId.IsValid() {
				defaultValues =
					append(defaultValues, DefaultValue{Id: Snowflake(mention.roleId), Type: "role"})
			}
		}
	}

	msg.DefaultValues = defaultValues

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
}

type ChannelSelectComponent struct {
	// CustomID is the custom unique ID.
	CustomID ComponentID `json:"custom_id,omitempty"`
	// ChannelTypes is the types of channels that can be chosen from.
	ChannelTypes []ChannelType `json:"channel_types,omitempty"`
	// Placeholder is the custom placeholder text if nothing is selected. Max
	// 100 characters.
	Placeholder string `json:"placeholder,omitempty"`
	// DefaultChannels is the list of channels that are marked as selected by default.
	DefaultChannels []ChannelID `json:"-"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Required dictates whether or not the user must fill out the component
	Required bool `json:"required"`
	// Disabled disables the select if true.
	Disabled bool `json:"disabled,omitempty"`
	// IDs of the selected channels
	Values []ChannelID `json:"values,omitempty"`
}

// ID implements the Component interface.
func (s *ChannelSelectComponent) ID() ComponentID { return s.CustomID }

// Type implements the Component interface.
func (s *ChannelSelectComponent) Type() ComponentType {
	return ChannelSelectComponentType
}

func (s *ChannelSelectComponent) _cmp() {}
func (s *ChannelSelectComponent) _icp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *ChannelSelectComponent) MarshalJSON() ([]byte, error) {
	type sel ChannelSelectComponent

	type DefaultValue struct {
		Id   ChannelID `json:"id"`
		Type string    `json:"type"`
	}

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
		MinValues     *int           `json:"min_values,omitempty"`
		MaxValues     *int           `json:"max_values,omitempty"`
		DefaultValues []DefaultValue `json:"default_values,omitempty"`
	}

	msg := Msg{
		Type: ChannelSelectComponentType,
		sel:  (*sel)(s),
	}

	var defaultValues []DefaultValue

	if len(s.DefaultChannels) > 0 {
		for _, channelId := range s.DefaultChannels {
			defaultValues = append(defaultValues, DefaultValue{Id: channelId, Type: "channel"})
		}
	}

	msg.DefaultValues = defaultValues

	if s.ValueLimits != [2]int{0, 0} {
		msg.MinValues = new(int)
		msg.MaxValues = new(int)

		*msg.MinValues = s.ValueLimits[0]
		*msg.MaxValues = s.ValueLimits[1]
	}

	return json.Marshal(msg)
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
func (u *UnknownComponent) _tlc() {}

type SectionComponent struct {
	// One to three child components representing the content of the section that is contextually associated to the accessory
	// Allowed values are TextDisplayComponent
	Components []Component `json:"components"`
	// A component that is contextually associated to the content of the section
	// Allowed values are ButtonComponent or ThumbnailComponent
	Accessory Component `json:"accessory"`
}

// Type implements the Component interface.
func (s *SectionComponent) Type() ComponentType {
	return SectionComponentType
}

func (s *SectionComponent) _cmp() {}
func (s *SectionComponent) _tlc() {}

// UnmarshalJSON unmarshals the section and parses nested component unions.
func (s *SectionComponent) UnmarshalJSON(b []byte) error {
	var section struct {
		Components []json.Raw `json:"components"`
		Accessory  json.Raw   `json:"accessory"`
	}

	if err := json.Unmarshal(b, &section); err != nil {
		return err
	}

	s.Components = make([]Component, len(section.Components))
	for i, raw := range section.Components {
		component, err := ParseComponent(raw)
		if err != nil {
			return fmt.Errorf("failed to parse section component %d: %w", i, err)
		}
		s.Components[i] = component
	}

	if len(section.Accessory) > 0 && string(section.Accessory) != "null" {
		accessory, err := ParseComponent(section.Accessory)
		if err != nil {
			return fmt.Errorf("failed to parse section accessory: %w", err)
		}
		s.Accessory = accessory
	}

	return nil
}

// MarshalJSON marshals the select in the format Discord expects.
func (s *SectionComponent) MarshalJSON() ([]byte, error) {
	type sel SectionComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: SectionComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type TextDisplayComponent struct {
	// Text that will be displayed similar to a message
	Content string `json:"content"`
}

// Type implements the Component interface.
func (s *TextDisplayComponent) Type() ComponentType {
	return TextDisplayComponentType
}

func (s *TextDisplayComponent) _cmp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *TextDisplayComponent) MarshalJSON() ([]byte, error) {
	type sel TextDisplayComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: TextDisplayComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type UnfurledMediaitem struct {
	// 	Supports arbitrary urls and attachment://<filename> references
	URL string `json:"url"`
	// 	The proxied url of the media item. This field is ignored and provided by the API as part of the response
	ProxyURL string `json:"proxy_url,omitempty"`
	// The height of the media item. This field is ignored and provided by the API as part of the response
	Height int `json:"height,omitempty"`
	// 	The width of the media item. This field is ignored and provided by the API as part of the response
	Width int `json:"width,omitempty"`
	// 	The media type of the content. This field is ignored and provided by the API as part of the response
	ContentType string `json:"content_type,omitempty"`
	// The id of the uploaded attachment. This field is ignored and provided by the API as part of the response
	// Only present if the media item was uploaded as an attachment.
	AttachmentID AttachmentID `json:"attachment_id,omitempty"`
}

type ThumbnailComponent struct {
	// A url or attachment provided as an [unfurled media item](/docs/components/reference#unfurled-media-item)
	Media UnfurledMediaitem `json:"media"`
	// Alt text for the media, max 1024 character
	Description string `json:"description,omitempty"`
	// Whether the thumbnail should be a spoiler (or blurred out). Defaults to `false`
	Spoiler bool `json:"spoiler,omitempty"`
}

// Type implements the Component interface.
func (s *ThumbnailComponent) Type() ComponentType {
	return ThumbnailComponentType
}

func (s *ThumbnailComponent) _cmp() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *ThumbnailComponent) MarshalJSON() ([]byte, error) {
	type sel ThumbnailComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: ThumbnailComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type MediaGalleryComponentItem struct {
	// A url or attachment provided as an unfurled media item
	Media UnfurledMediaitem `json:"media"`
	// Alt text for the media, max 1024 characters
	Description string `json:"description,omitempty"`
	// Whether the media should be a spoiler (or blurred out). Defaults to false
	Spoiler bool `json:"spoiler,omitempty"`
}

type MediaGalleryComponent struct {
	// 1 to 10 media gallery items
	Items []MediaGalleryComponentItem `json:"items"`
}

// Type implements the Component interface.
func (s *MediaGalleryComponent) Type() ComponentType {
	return MediaGalleryComponentType
}

func (s *MediaGalleryComponent) _cmp() {}
func (s *MediaGalleryComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *MediaGalleryComponent) MarshalJSON() ([]byte, error) {
	type sel MediaGalleryComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: MediaGalleryComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type FileComponent struct {
	// This unfurled media item is unique in that it **only** supports attachment references using the `attachment://<filename>` syntax
	File UnfurledMediaitem `json:"file"`
	// Whether the media should be a spoiler (or blurred out). Defaults to `false`
	Spoiler bool `json:"spoiler,omitempty"`
	// The name of the file. This field is ignored and provided by the API as part of the response
	Name string `json:"name,omitempty"`
	// The size of the file in bytes. This field is ignored and provided by the API as part of the response
	Size int `json:"size,omitempty"`
}

// Type implements the Component interface.
func (s *FileComponent) Type() ComponentType {
	return FileComponentType
}

func (s *FileComponent) _cmp() {}
func (s *FileComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *FileComponent) MarshalJSON() ([]byte, error) {
	type sel FileComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: FileComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type SeparatorComponentSpacing int

const (
	SeparatorComponentSpacingSmallPadding SeparatorComponentSpacing = iota + 1
	SeparatorComponentSpacingLargePadding
)

type SeparatorComponent struct {
	// Whether a visual divider should be displayed in the component. Defaults to `true`
	Divider bool `json:"divider,omitempty"`
	// Size of separator padding—`1` for small padding, `2` for large padding. Defaults to `
	Spacing SeparatorComponentSpacing `json:"spacing,omitempty"`
}

// Type implements the Component interface.
func (s *SeparatorComponent) Type() ComponentType {
	return SeparatorComponentType
}

func (s *SeparatorComponent) _cmp() {}
func (s *SeparatorComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *SeparatorComponent) MarshalJSON() ([]byte, error) {
	type sel SeparatorComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: SeparatorComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type ContainerComponent struct {
	// Child components that are encapsulated within the Container
	Components []Component `json:"components"`
	// Color for the accent on the container as RGB from `0x000000` to `0xFFFFFF`
	AccentColor Color `json:"accent_color,omitempty"`
	// Whether the container should be a spoiler (or blurred out). Defaults to `false`.
	Spoiler bool `json:"spoiler,omitempty"`
}

// Type implements the Component interface.
func (s *ContainerComponent) Type() ComponentType {
	return ContainerComponentType
}

func (s *ContainerComponent) _cmp() {}
func (s *ContainerComponent) _tlc() {}

// UnmarshalJSON unmarshals the container and parses nested component unions.
func (s *ContainerComponent) UnmarshalJSON(b []byte) error {
	var container struct {
		Components  []json.Raw `json:"components"`
		AccentColor Color      `json:"accent_color,omitempty"`
		Spoiler     bool       `json:"spoiler,omitempty"`
	}

	if err := json.Unmarshal(b, &container); err != nil {
		return err
	}

	s.Components = make([]Component, len(container.Components))
	for i, raw := range container.Components {
		component, err := ParseComponent(raw)
		if err != nil {
			return fmt.Errorf("failed to parse container component %d: %w", i, err)
		}
		s.Components[i] = component
	}

	s.AccentColor = container.AccentColor
	s.Spoiler = container.Spoiler

	return nil
}

// MarshalJSON marshals the select in the format Discord expects.
func (s *ContainerComponent) MarshalJSON() ([]byte, error) {
	type sel ContainerComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: ContainerComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type LabelComponent struct {
	// The label text; max 45 characters
	Label string `json:"label"`
	// An optional description text for the label; max 100 characters
	Description string `json:"description,omitempty"`
	// The component within the label
	Component Component `json:"component"`
}

// Type implements the Component interface.
func (s *LabelComponent) Type() ComponentType {
	return LabelComponentType
}

func (s *LabelComponent) _cmp() {}
func (s *LabelComponent) _tlc() {}

// UnmarshalJSON unmarshals the label and parses the nested component union.
func (s *LabelComponent) UnmarshalJSON(b []byte) error {
	var label struct {
		Label       string   `json:"label"`
		Description string   `json:"description,omitempty"`
		Component   json.Raw `json:"component"`
	}

	if err := json.Unmarshal(b, &label); err != nil {
		return err
	}

	s.Label = label.Label
	s.Description = label.Description

	if len(label.Component) > 0 && string(label.Component) != "null" {
		component, err := ParseComponent(label.Component)
		if err != nil {
			return fmt.Errorf("failed to parse label component: %w", err)
		}
		s.Component = component
	}

	return nil
}

// MarshalJSON marshals the select in the format Discord expects.
func (s *LabelComponent) MarshalJSON() ([]byte, error) {
	type sel LabelComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: LabelComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type FileUploadComponent struct {
	// ID for the file upload; 1-100 characters
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Minimum and maximum number of items that must be uploaded (defaults to 1); min 0, max 10
	ValueLimits [2]int `json:"-"`
	// Whether the file upload requires files to be uploaded before submitting the modal (defaults to `true`)
	Required bool `json:"required,omitempty"`
	// IDs of the uploaded files found in the [resolved data](/docs/interactions/receiving-and-responding#interaction-object-resolved-data-structure)
	Values []Snowflake `json:"values,omitempty"`
}

// Type implements the Component interface.
func (s *FileUploadComponent) Type() ComponentType {
	return FileUploadComponentType
}

func (s *FileUploadComponent) _cmp() {}
func (s *FileUploadComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *FileUploadComponent) MarshalJSON() ([]byte, error) {
	type sel FileUploadComponent

	type Msg struct {
		Type      ComponentType `json:"type"`
		MinValues *int          `json:"min_values,omitempty"`
		MaxValues *int          `json:"max_values,omitempty"`
		*sel
	}

	msg := Msg{
		Type: FileUploadComponentType,
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

type RadioGroupComponentOption struct {
	// Dev-defined value of the option; max 100 characters
	Value string `json:"value"`
	// User-facing label of the option; max 100 characters
	Label string `json:"label"`
	// Optional description for the option; max 100 characters
	Description string `json:"description,omitempty"`
	// Shows the option as selected by default
	Default bool `json:"default,omitempty"`
}

type RadioGroupComponent struct {
	// ID for the file upload; 1-100 characters
	CustomID ComponentID `json:"custom_id,omitempty"`
	// List of options to show; min 2, max 10
	Options []RadioGroupComponentOption `json:"options"`
	// Whether the file upload requires files to be uploaded before submitting the modal (defaults to `true`)
	Required bool `json:"required,omitempty"`
	// The value of the selected option, or null if no option is selected
	Value string `json:"value,omitempty"`
}

// Type implements the Component interface.
func (s *RadioGroupComponent) Type() ComponentType {
	return RadioGroupComponentType
}

func (b *RadioGroupComponent) ID() ComponentID { return b.CustomID }

func (s *RadioGroupComponent) _cmp() {}
func (s *RadioGroupComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *RadioGroupComponent) MarshalJSON() ([]byte, error) {
	type sel RadioGroupComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: RadioGroupComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}

type CheckboxGroupComponentOption struct {
	// Dev-defined value of the option; max 100 characters
	Value string `json:"value"`
	// User-facing label of the option; max 100 characters
	Label string `json:"label"`
	// Optional description for the option; max 100 characters
	Description string `json:"description,omitempty"`
	// Shows the option as selected by default
	Default bool `json:"default,omitempty"`
}

type CheckboxGroupComponent struct {
	// ID for the file upload; 1-100 characters
	CustomID ComponentID `json:"custom_id,omitempty"`
	// List of options to show; min 1, max 10
	Options []CheckboxGroupComponentOption `json:"options"`
	// ValueLimits is the minimum and maximum number of items that can be
	// chosen. The default is [1, 1] if ValueLimits is a zero-value.
	ValueLimits [2]int `json:"-"`
	// Whether the file upload requires files to be uploaded before submitting the modal (defaults to `true`)
	Required bool `json:"required,omitempty"`
	// The values of the selected options.
	Values []string `json:"values,omitempty"`
}

// Type implements the Component interface.
func (s *CheckboxGroupComponent) Type() ComponentType {
	return CheckboxGroupComponentType
}

func (s *CheckboxGroupComponent) _cmp() {}
func (s *CheckboxGroupComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *CheckboxGroupComponent) MarshalJSON() ([]byte, error) {
	type sel CheckboxGroupComponent

	type Msg struct {
		Type      ComponentType `json:"type"`
		MinValues *int          `json:"min_values,omitempty"`
		MaxValues *int          `json:"max_values,omitempty"`
		*sel
	}

	msg := Msg{
		Type: CheckboxGroupComponentType,
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

type CheckboxComponent struct {
	// ID for the file upload; 1-100 characters
	CustomID ComponentID `json:"custom_id,omitempty"`
	// Whether the checkbox is selected by default
	Default bool `json:"default,omitempty"`
	// The value of the selected option, or null if no option is selected
	Value bool `json:"value,omitempty"`
}

// Type implements the Component interface.
func (s *CheckboxComponent) Type() ComponentType {
	return CheckboxComponentType
}

func (s *CheckboxComponent) _cmp() {}
func (s *CheckboxComponent) _tlc() {}

// MarshalJSON marshals the select in the format Discord expects.
func (s *CheckboxComponent) MarshalJSON() ([]byte, error) {
	type sel CheckboxComponent

	type Msg struct {
		Type ComponentType `json:"type"`
		*sel
	}

	msg := Msg{
		Type: CheckboxComponentType,
		sel:  (*sel)(s),
	}

	return json.Marshal(msg)
}
