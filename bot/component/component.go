package component

import "github.com/diamondburned/arikawa/v3/discord"

// Component describes a bot component.
type Component interface {
	ToComponent() discord.Component
}

// Row describes a row of components.
type Row []Component

// Select describes a select component, which shows the user a list of choices
// to select from.
type Select struct {
	Options     []Option
	Placeholder string
	NSelections [2]int // default [1, 1], format [min, max]
	Disabled    bool
}

// Option is a select option.
type Option struct {
	Label       string
	Description string
	Emoji       discord.Emoji // ID, Name and Animated
	Default     bool
}

// Button is a clickable button.
type Button struct {
	Style discord.ButtonStyle
	Label string
}
