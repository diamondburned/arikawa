package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/bot"
	"github.com/diamondburned/arikawa/v3/utils/bot/extras/middlewares"
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

	// Those methods can take in a regular Go method reference.
	sub.ChangeCommandInfo(d.GOOS, "GOOS", "Prints the current operating system")
	sub.ChangeCommandInfo(d.GC, "GC", "Triggers the garbage collector")
	// They could also take in the raw name.
	sub.ChangeCommandInfo("Goroutines", "", "Prints the current number of Goroutines")

	sub.Hide(d.Die)
	sub.AddMiddleware(d.Die, middlewares.AdminOnly(d.Context))
}

// ~go goroutines
func (d *Debug) Goroutines(*gateway.MessageCreateEvent) (string, error) {
	return fmt.Sprintf(
		"goroutines: %d",
		runtime.NumGoroutine(),
	), nil
}

// ~go GOOS
func (d *Debug) GOOS(*gateway.MessageCreateEvent) (string, error) {
	return strings.Title(runtime.GOOS), nil
}

// ~go GC
func (d *Debug) GC(*gateway.MessageCreateEvent) (string, error) {
	runtime.GC()
	return "Done.", nil
}

// ~go die
// This command will be hidden from ~help by default.
func (d *Debug) Die(m *gateway.MessageCreateEvent) error {
	log.Fatalln("User", m.Author.Username, "killed the bot x_x")
	return nil
}
