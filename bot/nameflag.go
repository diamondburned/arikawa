package bot

import "strings"

type NameFlag uint64

const FlagSeparator = 'ー'

const (
	None NameFlag = 1 << iota

	// !!!
	//
	// These flags are applied to all events, if possible. The defined behavior
	// is to search for "ChannelID" fields or "ID" fields in structs with
	// "Channel" in its name. It doesn't handle individual events, as such, will
	// not be able to guarantee it will always work.

	// R - Raw, which tells the library to use the method name as-is (flags will
	// still be stripped). For example, if a method is called Reset its
	// command will also be Reset, without being all lower-cased.
	Raw

	// A - AdminOnly, which tells the library to only run the Subcommand/method
	// if the user is admin or not. This will automatically add GuildOnly as
	// well.
	AdminOnly

	// G - GuildOnly, which tells the library to only run the Subcommand/method
	// if the user is inside a guild.
	GuildOnly

	// M - Middleware, which tells the library that the method is a middleware.
	// The method will be executed anytime a method of the same struct is
	// matched.
	//
	// Using this flag inside the subcommand will drop all methods (this is an
	// undefined behavior/UB).
	Middleware

	// H - Hidden, which tells the router to not add this into the list of
	// commands, hiding it from Help. Handlers that are hidden will not have any
	// arguments parsed. It will be treated as an Event.
	Hidden
)

func ParseFlag(name string) (NameFlag, string) {
	parts := strings.SplitN(name, string(FlagSeparator), 2)
	if len(parts) != 2 {
		return 0, name
	}

	var f NameFlag

	for _, r := range parts[0] {
		switch r {
		case 'R':
			f |= Raw
		case 'A':
			f |= AdminOnly | GuildOnly
		case 'G':
			f |= GuildOnly
		case 'M':
			f |= Middleware
		case 'H':
			f |= Hidden
		}
	}

	return f, parts[1]
}

func (f NameFlag) Is(flag NameFlag) bool {
	return f&flag != 0
}
