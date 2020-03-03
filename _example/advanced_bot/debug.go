package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/diamondburned/arikawa/bot"
	"github.com/diamondburned/arikawa/gateway"
)

// Flag for administrators only.
type Debug struct {
	Context *bot.Context
}

// Setup demonstrates the CanSetup interface. This function will never be parsed
// as a callback of any event.
func (d *Debug) Setup(sub *bot.Subcommand) {
	// Set a custom command (e.g. "!go ..."):
	sub.Command = "go"
	// Set a custom description:
	sub.Description = "Print Go debugging variables"

	// Manually set the usage for each function.

	sub.ChangeCommandInfo("GOOS", "",
		"Prints the current operating system")

	sub.ChangeCommandInfo("GC", "",
		"Triggers the garbage collecto")

	sub.ChangeCommandInfo("Goroutines", "",
		"Prints the current number of Goroutines")
}

// ~go goroutines
func (d *Debug) Goroutines(m *gateway.MessageCreateEvent) (string, error) {
	return fmt.Sprintf(
		"goroutines: %d",
		runtime.NumGoroutine(),
	), nil
}

// ~go GOOS
func (d *Debug) R_GOOS(m *gateway.MessageCreateEvent) (string, error) {
	return strings.Title(runtime.GOOS), nil
}

// ~go GC
func (d *Debug) R_GC(m *gateway.MessageCreateEvent) (string, error) {
	runtime.GC()
	return "Done.", nil
}

// ~go die
// This command will be hidden from ~help by default.
func (d *Debug) A_Die(m *gateway.MessageCreateEvent) error {
	log.Fatalln("User", m.Author.Username, "killed the bot x_x")
	return nil
}
