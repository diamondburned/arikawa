package discord

import "time"

type Command struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Options     []CommandOption `json:"options,omitempty"`
	ID          CommandID       `json:"id"`
	AppID       AppID           `json:"application_id"`
}

// CreatedAt returns a time object representing when the command was created.
func (c Command) CreatedAt() time.Time {
	return c.ID.Time()
}

type CommandOption struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Choices     []CommandOptionChoice `json:"choices,omitempty"`
	Options     []CommandOption       `json:"options,omitempty"`
	Type        CommandOptionType     `json:"type"`
	Required    bool                  `json:"required"`
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
)

type CommandOptionChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
