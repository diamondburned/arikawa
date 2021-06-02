// Package main demonstrates an advanced bot that uses the bot router library to
// make commands.
package main

import (
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/bot"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	commands := &Bot{}

	bot.Run(token, commands, func(ctx *bot.Context) error {
		ctx.HasPrefix = bot.NewPrefix("!", "~")
		ctx.EditableCommands = true

		// Subcommand demo, but this can be in another package.
		ctx.MustRegisterSubcommand(&Debug{})

		// The bot package will automatically derive out Gateway intents. It
		// might not catch everything though, so a ctx.Gateway.AddIntents is
		// always available.

		return nil
	})
}
