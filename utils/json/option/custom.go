package option

import "github.com/diamondburned/arikawa/discord"

// ================================ Seconds ================================

// Seconds is the option type for discord.Seconds.
type Seconds = *discord.Seconds

// ZeroSeconds are 0 Seconds.
var ZeroSeconds = NewSeconds(0)

// NewString creates a new Seconds with the value of the passed discord.Seconds.
func NewSeconds(s discord.Seconds) Seconds { return &s }

// ================================ Color ================================

// Color is the option type for discord.Color.
type Color = *discord.Color

// NewString creates a new Color with the value of the passed discord.Color.
func NewColor(s discord.Color) Color { return &s }
