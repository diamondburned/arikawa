package bot

import "strings"

type NameFlag uint64

var FlagSeparator = 'ー'

const None NameFlag = 0

// !!!
//
// These flags are applied to all events, if possible. The defined behavior
// is to search for "ChannelID" fields or "ID" fields in structs with
// "Channel" in its name. It doesn't handle individual events, as such, will
// not be able to guarantee it will always work. Refer to package infer.

// R - Raw, which tells the library to use the method name as-is (flags will
// still be stripped). For example, if a method is called Reset its
// command will also be Reset, without being all lower-cased.
const Raw NameFlag = 1 << 1

// A - AdminOnly, which tells the library to only run the Subcommand/method
// if the user is admin or not. This will automatically add GuildOnly as
// well.
const AdminOnly NameFlag = 1 << 2

// G - GuildOnly, which tells the library to only run the Subcommand/method
// if the user is inside a guild.
const GuildOnly NameFlag = 1 << 3

// M - Middleware, which tells the library that the method is a middleware.
// The method will be executed anytime a method of the same struct is
// matched.
//
// Using this flag inside the subcommand will drop all methods (this is an
// undefined behavior/UB).
const Middleware NameFlag = 1 << 4

// H - Hidden/Handler, which tells the router to not add this into the list
// of commands, hiding it from Help. Handlers that are hidden will not have
// any arguments parsed. It will be treated as an Event.
const Hidden NameFlag = 1 << 5

// P - Plumb, which tells the router to call only this handler with all the
// arguments (except the prefix string). If plumb is used, only this method
// will be called for the given struct, though all other events as well as
// methods with the H (Hidden/Handler) flag.
//
// This is different from using H (Hidden/Handler), as handlers are called
// regardless of command prefixes. Plumb methods are only called once, and
// no other methods will be called for that struct. That said, a Plumb
// method would still go into Commands, but only itself will be there.
//
// Note that if there's a Plumb method in the main commands, then none of
// the subcommands would be called. This is an unintended but expected side
// effect.
//
// Example
//
// A use for this would be subcommands that don't need a second command, or
// if the main struct manually handles command switching. This example
// demonstrates the second use-case:
//
//    func (s *Sub) PーMain(
//        c *gateway.MessageCreateGateway, c *Content) error {
//
//        // Input:  !sub this is a command
//        // Output: this is a command
//
//        log.Println(c.String())
//        return nil
//    }
//
const Plumb NameFlag = 1 << 6

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
		case 'P':
			f |= Plumb
		}
	}

	return f, parts[1]
}

func (f NameFlag) Is(flag NameFlag) bool {
	return f&flag != 0
}
