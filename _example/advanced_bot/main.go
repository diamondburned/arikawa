package main

import (
	"log"
	"os"

	"github.com/diamondburned/arikawa/bot"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	commands := &Bot{}

	wait, err := bot.Start(token, commands, func(ctx *bot.Context) error {
		ctx.Prefix = "!"

		// Subcommand demo, but this can be in another package.
		ctx.MustRegisterSubcommand(&Debug{})

		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Bot started")

	// Wait is the same as bot.Wait().
	wait()
}
