package discord

import "time"

type Command struct {
	ID          CommandID       `json:"id"`
	AppID       AppID           `json:"application_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Options     []CommandOption `json:"options,omitempty"`
}

// CreatedAt returns a time object representing when the command was created.
func (c Command) CreatedAt() time.Time {
	return c.ID.Time()
}

type CommandOption struct {
	Type        CommandOptionType     `json:"type"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Required    bool                  `json:"required"`
	Choices     []CommandOptionChoice `json:"choices,omitempty"`
	Options     []CommandOption       `json:"options,omitempty"`
}

type CommandOptionType uint

const (
	SubcommandOption CommandOptionType = iota + 1
	SubcommandGroupOption
	StringOption
	IntegerOption
	BooleanOption
	UserOption
	ChannelOption
	RoleOption
	MentionableOption
	NumberOption
)

type CommandOptionChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
