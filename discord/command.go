package discord

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// CommandType is the type of the command, which describes the intended
// invokation source of the command.
type CommandType uint

const (
	ChatInputCommand CommandType = iota + 1
	UserCommand
	MessageCommand
)

// Command is the base "command" model that belongs to an application. This is
// what you are creating when you POST a new command.
//
// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	// ID is the unique id of the command.
	ID CommandID `json:"id"`
	// Type is the intended source of the command.
	Type CommandType `json:"type,omitempty"`
	// AppID is the unique id of the parent application.
	AppID AppID `json:"application_id"`
	// GuildID is the guild id of the command, if not global.
	GuildID GuildID `json:"guild_id,omitempty"`
	// Name is the 1-32 lowercase character name matching ^[\w-]{1,32}$.
	Name              string        `json:"name"`
	NameLocalizations StringLocales `json:"name_localizations,omitempty"`
	// Description is the 1-100 character description.
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	// LocalizedName is only populated when this is received from Discord's API.
	LocalizedName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
	// Options are the parameters for the command. Its types are value types,
	// which can either be a SubcommandOption or a SubcommandGroupOption.
	//
	// Note that required options must be listed before optional options, and
	// a command, or each individual subcommand, can have a maximum of 25
	// options.
	//
	// It is only present on ChatInputCommands.
	Options CommandOptions `json:"options,omitempty"`
	// DefaultMemberPermissions is set of permissions.
	DefaultMemberPermissions *Permissions `json:"default_member_permissions,string,omitempty"`
	// NoDMPermission indicates whether the command is NOT available in DMs with
	// the app, only for globally-scoped commands. By default, commands are visible.
	NoDMPermission bool `json:"-"`
	// NoDefaultPermissions defines whether the command is NOT enabled by
	// default when the app is added to a guild.
	NoDefaultPermission bool `json:"-"`
	// Version is an autoincrementing version identifier updated during
	// substantial record changes
	Version Snowflake `json:"version,omitempty"`
}

// Language is a string type for language codes, such as "en-US" or "fr". Refer
// to the constants for valid language codes.
//
// The list of all valid language codes are at
// https://discord.com/developers/docs/reference#locales
type Language string

// StringLocales is the map mapping a language code to a localized string.
type StringLocales map[Language]string

const (
	Danish        Language = "da"
	German        Language = "de"
	EnglishUK     Language = "en-GB"
	EnglishUS     Language = "en-US"
	Spanish       Language = "es-ES"
	French        Language = "fr"
	Croatian      Language = "hr"
	Italian       Language = "it"
	Lithuanian    Language = "lt"
	Hungarian     Language = "hu"
	Dutch         Language = "nl"
	Norwegian     Language = "no"
	Polish        Language = "pl"
	PortugueseBR  Language = "pt-BR"
	Romanian      Language = "ro"
	Finnish       Language = "fi"
	Swedish       Language = "sv-SE"
	Vietnamese    Language = "vi"
	Turkish       Language = "tr"
	Czech         Language = "cs"
	Greek         Language = "el"
	Bulgarian     Language = "bg"
	Russian       Language = "ru"
	Ukrainian     Language = "uk"
	Hindi         Language = "hi"
	Thai          Language = "th"
	ChineseChina  Language = "zh-CN"
	Japanese      Language = "ja"
	ChineseTaiwan Language = "zh-TW"
	Korean        Language = "ko"
)

// CreatedAt returns a time object representing when the command was created.
func (c *Command) CreatedAt() time.Time {
	return c.ID.Time()
}

func (c *Command) MarshalJSON() ([]byte, error) {
	type RawCommand Command
	cmd := struct {
		*RawCommand
		DMPermission      bool `json:"dm_permission"`
		DefaultPermission bool `json:"default_permission"`
	}{RawCommand: (*RawCommand)(c)}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	cmd.DefaultPermission = !c.NoDefaultPermission
	cmd.DMPermission = !c.NoDMPermission

	return json.Marshal(cmd)
}

func (c *Command) UnmarshalJSON(data []byte) error {
	type rawCommand Command

	cmd := struct {
		*rawCommand
		DMPermission      bool `json:"dm_permission"`
		DefaultPermission bool `json:"default_permission"`
	}{
		rawCommand: (*rawCommand)(c),
	}

	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	// Discord defaults default_permission to true, so we need to invert the
	// meaning of the field (>No<DefaultPermission) to match Go's default
	// value, false.
	c.NoDefaultPermission = !cmd.DefaultPermission
	c.NoDMPermission = !cmd.DMPermission

	// Discord defaults type to 1 if omitted.
	if c.Type == 0 {
		c.Type = ChatInputCommand
	}

	return nil
}

// commandTypeCheckError is returned if a one of Command's Options fails the
// type check.
type commandTypeCheckError struct {
	name   string
	got    interface{}
	expect string
}

// Name returns the name of the erroneous command.
func (err commandTypeCheckError) Name() string {
	return err.name
}

// Data returns the erroneous data that belongs to this error. It is usually
// either a CommandOption or a CommandOptionValue.
func (err commandTypeCheckError) Data() interface{} {
	return err.got
}

// Error implements error.
func (err commandTypeCheckError) Error() string {
	return fmt.Sprintf(
		"error at option name %q: expected %s, got %T",
		err.name, err.expect, err.got,
	)
}

// CommandOptions is used primarily for unmarshaling.
type CommandOptions []CommandOption

// UnmarshalJSON unmarshals b into these CommandOptions.
func (c *CommandOptions) UnmarshalJSON(b []byte) error {
	var unknowns []UnknownCommandOption
	if err := json.Unmarshal(b, &unknowns); err != nil {
		return err
	}

	if len(unknowns) == 0 {
		*c = nil
		return nil
	}

	*c = make([]CommandOption, len(unknowns))
	for i, v := range unknowns {
		(*c)[i] = v.data
	}

	return nil
}

// UnknownCommandOption is used for unknown or unmarshaled CommandOption values.
// It is used in the unmarshaling stage for all CommandOption types.
//
// An UnknownCommandOption will satisfy both CommandOption and
// CommandOptionValue. Code that type-switches on either of them should not
// assume that only the expected types are used.
type UnknownCommandOption struct {
	OptionName string            `json:"name"`
	OptionType CommandOptionType `json:"type"`

	raw  json.Raw
	data CommandOption
}

// Name returns the supposeed name for this UnknownCommandOption.
func (u *UnknownCommandOption) Name() string {
	return u.OptionName
}

// Type returns the supposed type for this UnknownCommandOption.
func (u *UnknownCommandOption) Type() CommandOptionType {
	return u.OptionType
}

// Raw returns the raw JSON of this UnknownCommandOption. It will only return a
// non-nil blob of JSON if the command option's type cannot be found. If this
// method doesn't return nil, then Data's type will be UnknownCommandOption.
func (u *UnknownCommandOption) Raw() json.Raw {
	return u.raw
}

// Data returns the underlying data type, which is a type that satisfies either
// CommandOption or CommandOptionValue.
func (u *UnknownCommandOption) Data() CommandOption {
	return u.data
}

// Implement both CommandOption and CommandOptionValue.
func (u *UnknownCommandOption) _val() {}

// UnmarshalJSON parses the JSON into the struct as-is then reads all its
// children Options/Choices (if subcommand(group)). Typed command options are
// created into u.Data, or u.Raw if the type is unknown. This is done from the
// bottom up.
func (u *UnknownCommandOption) UnmarshalJSON(b []byte) error {
	type unknown UnknownCommandOption

	if err := json.Unmarshal(b, (*unknown)(u)); err != nil {
		return fmt.Errorf("failed to unmarshal unknown: %w", err)
	}

	switch u.Type() {
	case SubcommandOptionType:
		u.data = &SubcommandOption{}
	case SubcommandGroupOptionType:
		u.data = &SubcommandGroupOption{}
	case StringOptionType:
		u.data = &StringOption{}
	case IntegerOptionType:
		u.data = &IntegerOption{}
	case BooleanOptionType:
		u.data = &BooleanOption{}
	case UserOptionType:
		u.data = &UserOption{}
	case ChannelOptionType:
		u.data = &ChannelOption{}
	case RoleOptionType:
		u.data = &RoleOption{}
	case MentionableOptionType:
		u.data = &MentionableOption{}
	case NumberOptionType:
		u.data = &NumberOption{}
	default:
		// Copy the blob of bytes into a new slice.
		u.raw = append(json.Raw(nil), b...)
		u.data = u
		return nil
	}

	if err := json.Unmarshal(b, u.data); err != nil {
		return fmt.Errorf("failed to unmarshal type %d: %w", u.Type(), err)
	}

	return nil
}

// CommandOptionType is the enumerated integer type for command options. The
// user usually won't have to touch any of these enum constants.
type CommandOptionType uint

const (
	SubcommandOptionType CommandOptionType = iota + 1
	SubcommandGroupOptionType
	StringOptionType
	IntegerOptionType
	BooleanOptionType
	UserOptionType
	ChannelOptionType
	RoleOptionType
	MentionableOptionType
	NumberOptionType
	AttachmentOptionType
	maxOptionType // for bound checking
)

// CommandOption is a union of command option types. The constructors for
// CommandOption will hint the types that can be a CommandOption.
//
// The following types implement this interface:
//
//   - *SubcommandGroupOption
//   - *SubcommandOption
//   - *StringOption
//   - *IntegerOption
//   - *BooleanOption
//   - *UserOption
//   - *ChannelOption
//   - *RoleOption
//   - *MentionableOption
//   - *NumberOption
//   - *AttachmentOption
type CommandOption interface {
	Name() string
	Type() CommandOptionType
}

// Maintaining these structs is quite an effort. If a new field is added into
// the generic CommandOption type, you MUST update ALL CommandOption structs.
// This means copy-pasting, yes.

// SubcommandGroupOption is a subcommand group that fits into a CommandOption.
type SubcommandGroupOption struct {
	OptionName               string              `json:"name"`
	OptionNameLocalizations  StringLocales       `json:"name_localizations,omitempty"`
	Description              string              `json:"description"`
	DescriptionLocalizations StringLocales       `json:"description_localizations,omitempty"`
	Required                 bool                `json:"required"`
	Subcommands              []*SubcommandOption `json:"options"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (s *SubcommandGroupOption) Name() string { return s.OptionName }

// Type implements CommandOption.
func (s *SubcommandGroupOption) Type() CommandOptionType { return SubcommandGroupOptionType }

// SubcommandOption is a subcommand option that fits into a CommandOption.
type SubcommandOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// Options contains command option values. All CommandOption types except
	// for SubcommandOption and SubcommandGroupOption will implement this
	// interface.
	Options []CommandOptionValue `json:"options"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (s *SubcommandOption) Name() string { return s.OptionName }

// Type implements CommandOption.
func (s *SubcommandOption) Type() CommandOptionType { return SubcommandOptionType }

// UnmarshalJSON unmarshals the given JSON bytes. It actually does
// type-checking.
func (s *SubcommandOption) UnmarshalJSON(b []byte) error {
	type raw SubcommandOption

	var opt struct {
		*raw
		Type    CommandOptionType      `json:"type"`
		Options []UnknownCommandOption `json:"options"`
	}

	opt.raw = (*raw)(s)

	if err := json.Unmarshal(b, &opt); err != nil {
		return err
	}

	if opt.Type != SubcommandOptionType {
		return fmt.Errorf("unexpected (not SubcommandOption) type %d", s.Type())
	}

	s.Options = make([]CommandOptionValue, len(opt.Options))
	for i, opt := range opt.Options {
		ov, ok := opt.data.(CommandOptionValue)
		if !ok {
			return commandTypeCheckError{opt.OptionName, opt.data, "CommandOptionValue"}
		}
		s.Options[i] = ov
	}

	return nil
}

// CommandOptionValue is a subcommand option that fits into a subcommand.
//
// The following types implement this interface:
//
//   - *StringOption
//   - *IntegerOption
//   - *BooleanOption
//   - *UserOption
//   - *ChannelOption
//   - *RoleOption
//   - *MentionableOption
//   - *NumberOption
//   - *AttachmentOption
type CommandOptionValue interface {
	CommandOption
	_val()
}

// StringOption is a subcommand option that fits into a CommandOptionValue.
type StringOption struct {
	OptionName               string         `json:"name"`
	OptionNameLocalizations  StringLocales  `json:"name_localizations,omitempty"`
	Description              string         `json:"description"`
	DescriptionLocalizations StringLocales  `json:"description_localizations,omitempty"`
	Required                 bool           `json:"required"`
	Choices                  []StringChoice `json:"choices,omitempty"`
	MinLength                option.Int     `json:"min_length,omitempty"`
	MaxLength                option.Int     `json:"max_length,omitempty"`
	// Autocomplete must not be true if Choices are present.
	Autocomplete bool `json:"autocomplete"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (s *StringOption) Name() string { return s.OptionName }

// Type implements CommandOptionValue.
func (s *StringOption) Type() CommandOptionType { return StringOptionType }
func (s *StringOption) _val()                   {}

// StringChoice is a pair of string key to a string.
type StringChoice struct {
	Name              string        `json:"name"`
	NameLocalizations StringLocales `json:"name_localizations,omitempty"`
	Value             string        `json:"value"`
	// LocalizedName is only populated when this is received from Discord's API.
	LocalizedName string `json:"name_localized,omitempty"`
}

// IntegerOption is a subcommand option that fits into a CommandOptionValue.
type IntegerOption struct {
	OptionName               string          `json:"name"`
	OptionNameLocalizations  StringLocales   `json:"name_localizations,omitempty"`
	Description              string          `json:"description"`
	DescriptionLocalizations StringLocales   `json:"description_localizations,omitempty"`
	Required                 bool            `json:"required"`
	Min                      option.Int      `json:"min_value,omitempty"`
	Max                      option.Int      `json:"max_value,omitempty"`
	Choices                  []IntegerChoice `json:"choices,omitempty"`
	// Autocomplete must not be true if Choices are present.
	Autocomplete bool `json:"autocomplete"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (i *IntegerOption) Name() string { return i.OptionName }

// Type implements CommandOptionValue.
func (i *IntegerOption) Type() CommandOptionType { return IntegerOptionType }
func (i *IntegerOption) _val()                   {}

// IntegerChoice is a pair of string key to an integer.
type IntegerChoice struct {
	Name              string        `json:"name"`
	NameLocalizations StringLocales `json:"name_localizations,omitempty"`
	Value             int           `json:"value"`
	// LocalizedName is only populated when this is received from Discord's API.
	LocalizedName string `json:"name_localized,omitempty"`
}

// BooleanOption is a subcommand option that fits into a CommandOptionValue.
type BooleanOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (b *BooleanOption) Name() string { return b.OptionName }

// Type implements CommandOptionValue.
func (b *BooleanOption) Type() CommandOptionType { return BooleanOptionType }
func (b *BooleanOption) _val()                   {}

// UserOption is a subcommand option that fits into a CommandOptionValue.
type UserOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (u *UserOption) Name() string { return u.OptionName }

// Type implements CommandOptionValue.
func (u *UserOption) Type() CommandOptionType { return UserOptionType }
func (u *UserOption) _val()                   {}

// ChannelOption is a subcommand option that fits into a CommandOptionValue.
type ChannelOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	ChannelTypes             []ChannelType `json:"channel_types,omitempty"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (c *ChannelOption) Name() string { return c.OptionName }

// Type implements CommandOptionValue.
func (c *ChannelOption) Type() CommandOptionType { return ChannelOptionType }
func (c *ChannelOption) _val()                   {}

// RoleOption is a subcommand option that fits into a CommandOptionValue.
type RoleOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (r *RoleOption) Name() string { return r.OptionName }

// Type implements CommandOptionValue.
func (r *RoleOption) Type() CommandOptionType { return RoleOptionType }
func (r *RoleOption) _val()                   {}

// MentionableOption is a subcommand option that fits into a CommandOptionValue.
type MentionableOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (m *MentionableOption) Name() string { return m.OptionName }

// Type implements CommandOptionValue.
func (m *MentionableOption) Type() CommandOptionType { return MentionableOptionType }
func (m *MentionableOption) _val()                   {}

// NumberOption is a subcommand option that fits into a CommandOptionValue.
type NumberOption struct {
	OptionName               string         `json:"name"`
	OptionNameLocalizations  StringLocales  `json:"name_localizations,omitempty"`
	Description              string         `json:"description"`
	DescriptionLocalizations StringLocales  `json:"description_localizations,omitempty"`
	Required                 bool           `json:"required"`
	Min                      option.Float   `json:"min_value,omitempty"`
	Max                      option.Float   `json:"max_value,omitempty"`
	Choices                  []NumberChoice `json:"choices,omitempty"`
	// Autocomplete must not be true if Choices are present.
	Autocomplete bool `json:"autocomplete"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (n *NumberOption) Name() string { return n.OptionName }

// Type implements CommandOptionValue.
func (n *NumberOption) Type() CommandOptionType { return NumberOptionType }
func (n *NumberOption) _val()                   {}

// NumberChoice is a pair of string key to a float64 values.
type NumberChoice struct {
	Name              string        `json:"name"`
	NameLocalizations StringLocales `json:"name_localizations,omitempty"`
	Value             float64       `json:"value"`
	// LocalizedName is only populated when this is received from Discord's API.
	LocalizedName string `json:"name_localized,omitempty"`
}

// AttachmentOption is a subcommand option that fits into a CommandOptionValue.
type AttachmentOption struct {
	OptionName               string        `json:"name"`
	OptionNameLocalizations  StringLocales `json:"name_localizations,omitempty"`
	Description              string        `json:"description"`
	DescriptionLocalizations StringLocales `json:"description_localizations,omitempty"`
	Required                 bool          `json:"required"`
	// LocalizedOptionName is only populated when this is received from
	// Discord's API.
	LocalizedOptionName string `json:"name_localized,omitempty"`
	// LocalizedDescription is only populated when this is received from
	// Discord's API.
	LocalizedDescription string `json:"description_localized,omitempty"`
}

// Name implements CommandOption.
func (n *AttachmentOption) Name() string { return n.OptionName }

// Type implements CommandOptionValue.
func (n *AttachmentOption) Type() CommandOptionType { return AttachmentOptionType }
func (n *AttachmentOption) _val()                   {}

// NewCommand creates a new command.
func NewCommand(name, description string, options ...CommandOption) Command {
	return Command{
		Name:        name,
		Description: description,
		Options:     options,
	}
}

// NewSubcommandGroupOption creates a new subcommand group option.
func NewSubcommandGroupOption(name, description string, subs ...*SubcommandOption) *SubcommandGroupOption {
	return &SubcommandGroupOption{
		OptionName:  name,
		Description: description,
		Subcommands: subs,
	}
}

// NewSubcommandOption creates a new subcommand option.
func NewSubcommandOption(name, description string, options ...CommandOptionValue) *SubcommandOption {
	return &SubcommandOption{
		OptionName:  name,
		Description: description,
		Options:     options,
	}
}

// NewStringOption creates a new string option.
func NewStringOption(name, description string, required bool) *StringOption {
	return &StringOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewIntegerOption creates a new integer option.
func NewIntegerOption(name, description string, required bool) *IntegerOption {
	return &IntegerOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewBooleanOption creates a new boolean option.
func NewBooleanOption(name, description string, required bool) *BooleanOption {
	return &BooleanOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewUserOption creates a new user option.
func NewUserOption(name, description string, required bool) *UserOption {
	return &UserOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewChannelOption creates a new channel option.
func NewChannelOption(name, description string, required bool) *ChannelOption {
	return &ChannelOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewRoleOption creates a new role option.
func NewRoleOption(name, description string, required bool) *RoleOption {
	return &RoleOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewMentionableOption creates a new mentionable option.
func NewMentionableOption(name, description string, required bool) *MentionableOption {
	return &MentionableOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// NewNumberOption creates a new number option.
func NewNumberOption(name, description string, required bool) *NumberOption {
	return &NumberOption{
		OptionName:  name,
		Description: description,
		Required:    required,
	}
}

// Generated with utils/generate-option-marshalers.sh

// MarshalJSON marshals SubcommandOption to JSON with the "type" field.
func (s *SubcommandOption) MarshalJSON() ([]byte, error) {
	type raw SubcommandOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: s.Type(),
		raw:  (*raw)(s),
	})
}

// MarshalJSON marshals SubcommandGroupOption to JSON with the "type" field.
func (s *SubcommandGroupOption) MarshalJSON() ([]byte, error) {
	type raw SubcommandGroupOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: s.Type(),
		raw:  (*raw)(s),
	})
}

// MarshalJSON marshals StringOption to JSON with the "type" field.
func (s *StringOption) MarshalJSON() ([]byte, error) {
	type raw StringOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: s.Type(),
		raw:  (*raw)(s),
	})
}

// MarshalJSON marshals IntegerOption to JSON with the "type" field.
func (i *IntegerOption) MarshalJSON() ([]byte, error) {
	type raw IntegerOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: i.Type(),
		raw:  (*raw)(i),
	})
}

// MarshalJSON marshals BooleanOption to JSON with the "type" field.
func (b *BooleanOption) MarshalJSON() ([]byte, error) {
	type raw BooleanOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: b.Type(),
		raw:  (*raw)(b),
	})
}

// MarshalJSON marshals UserOption to JSON with the "type" field.
func (u *UserOption) MarshalJSON() ([]byte, error) {
	type raw UserOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: u.Type(),
		raw:  (*raw)(u),
	})
}

// MarshalJSON marshals ChannelOption to JSON with the "type" field.
func (c *ChannelOption) MarshalJSON() ([]byte, error) {
	type raw ChannelOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: c.Type(),
		raw:  (*raw)(c),
	})
}

// MarshalJSON marshals RoleOption to JSON with the "type" field.
func (r *RoleOption) MarshalJSON() ([]byte, error) {
	type raw RoleOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: r.Type(),
		raw:  (*raw)(r),
	})
}

// MarshalJSON marshals MentionableOption to JSON with the "type" field.
func (m *MentionableOption) MarshalJSON() ([]byte, error) {
	type raw MentionableOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: m.Type(),
		raw:  (*raw)(m),
	})
}

// MarshalJSON marshals NumberOption to JSON with the "type" field.
func (n *NumberOption) MarshalJSON() ([]byte, error) {
	type raw NumberOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: n.Type(),
		raw:  (*raw)(n),
	})
}

// MarshalJSON marshals AttachmentOption to JSON with the "type" field.
func (a *AttachmentOption) MarshalJSON() ([]byte, error) {
	type raw AttachmentOption
	return json.Marshal(struct {
		Type CommandOptionType `json:"type"`
		*raw
	}{
		Type: a.Type(),
		raw:  (*raw)(a),
	})
}
