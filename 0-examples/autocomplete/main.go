package main

import (
	"context"
	"log"
	"os"
	"strings"

	"libdb.so/arikawa/v4/api"
	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/state"
	"libdb.so/arikawa/v4/utils/json/option"
)

// To run, do `GUILD_ID="GUILD ID" BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	guildID := discord.GuildID(mustSnowflakeEnv("GUILD_ID"))

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s := state.New("Bot " + token)

	app, err := s.CurrentApplication()
	if err != nil {
		log.Fatalln("Failed to get application ID:", err)
	}

	s.AddHandler(func(e *gateway.InteractionCreateEvent) {
		var resp api.InteractionResponse
		switch d := e.Data.(type) {
		case *discord.CommandInteraction:
			content := option.NewNullableString("Pong: " + d.Options[0].String() + "!")
			resp = api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: content,
				},
			}
		case *discord.AutocompleteInteraction:
			allChoices := api.AutocompleteStringChoices{
				{Name: "Choice A", Value: "Choice A"},
				{Name: "Choice B", Value: "Choice B"},
				{Name: "Choice C", Value: "Choice C"},
				{Name: "Abc Def", Value: "Abcdef"},
				{Name: "Ghi Jkl", Value: "Ghijkl"},
				{Name: "Mno Pqr", Value: "Mnopqr"},
				{Name: "Stu Vwx", Value: "Stuvwx"},
			}
			query := strings.ToLower(d.Options[0].String())
			var choices api.AutocompleteStringChoices
			for _, choice := range allChoices {
				if strings.HasPrefix(strings.ToLower(choice.Name), query) ||
					strings.HasPrefix(strings.ToLower(choice.Value), query) {
					choices = append(choices, choice)
				}
			}
			resp = api.InteractionResponse{
				Type: api.AutocompleteResult,
				Data: &api.InteractionResponseData{
					Choices: &choices,
				},
			}
		default:
			return
		}

		if err := s.RespondInteraction(e.ID, e.Token, resp); err != nil {
			log.Println("failed to send interaction callback:", err)
		}
	})

	s.AddIntents(gateway.IntentGuilds)
	s.AddIntents(gateway.IntentGuildMessages)

	if err := s.Open(context.Background()); err != nil {
		log.Fatalln("failed to open:", err)
	}
	defer s.Close()

	log.Println("Gateway connected. Getting all guild commands.")

	commands, err := s.GuildCommands(app.ID, guildID)
	if err != nil {
		log.Fatalln("failed to get guild commands:", err)
	}

	for _, command := range commands {
		log.Println("Existing command", command.Name, "found.")
	}

	newCommands := []api.CreateCommandData{
		{
			Name:        "ping",
			Description: "Basic ping command.",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:   "text",
					Description:  "Text to echo back",
					Autocomplete: true,
				},
			},
		},
	}

	if _, err := s.BulkOverwriteGuildCommands(app.ID, guildID, newCommands); err != nil {
		log.Fatalln("failed to create guild command:", err)
	}

	// Block forever.
	select {}
}

func mustSnowflakeEnv(env string) discord.Snowflake {
	s, err := discord.ParseSnowflake(os.Getenv(env))
	if err != nil {
		log.Fatalf("Invalid snowflake for $%s: %v", env, err)
	}
	return s
}
