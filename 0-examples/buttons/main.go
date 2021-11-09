package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// To run, do `GUILD_ID="GUILD ID" BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	guildID := discord.GuildID(mustSnowflakeEnv("GUILD_ID"))

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	s, err := session.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
		return
	}

	app, err := s.CurrentApplication()
	if err != nil {
		log.Fatalln("Failed to get application ID:", err)
	}
	appID := app.ID

	s.AddHandler(func(e *gateway.InteractionCreateEvent) {
		if e.Type == discord.CommandInteraction {
			// Send a message with a button back on slash commands.
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("This is a message with a button!"),
					Components: &[]discord.Component{
						&discord.ActionRowComponent{
							Components: []discord.Component{
								&discord.ButtonComponent{
									Label:    "Hello World!",
									CustomID: "first_button",
									Emoji: &discord.ButtonEmoji{
										Name: "👋",
									},
									Style: discord.PrimaryButton,
								},
								&discord.ButtonComponent{
									Label:    "Secondary",
									CustomID: "second_button",
									Style:    discord.SecondaryButton,
								},
								&discord.ButtonComponent{
									Label:    "Success",
									CustomID: "success_button",
									Style:    discord.SuccessButton,
								},
								&discord.ButtonComponent{
									Label:    "Danger",
									CustomID: "danger_button",
									Style:    discord.DangerButton,
								},
								&discord.ButtonComponent{
									Label: "Link",
									URL:   "https://google.com",
									Style: discord.LinkButton,
								},
							},
						},
					},
				},
			}

			if err := s.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback:", err)
			}
		}

		if e.Type != discord.ComponentInteraction {
			return
		}
		customID := e.Data.(*discord.ComponentInteractionData).CustomID
		data := api.InteractionResponse{
			Type: api.UpdateMessage,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString("Custom ID: " + customID),
			},
		}

		if err := s.RespondInteraction(e.ID, e.Token, data); err != nil {
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

	commands, err := s.GuildCommands(appID, guildID)
	if err != nil {
		log.Fatalln("failed to get guild commands:", err)
	}

	for _, command := range commands {
		log.Println("Existing command", command.Name, "found.")
	}

	newCommands := []api.CreateCommandData{
		{
			Name:        "buttons",
			Description: "Send an interactable message.",
		},
	}

	for _, command := range newCommands {
		_, err := s.CreateGuildCommand(appID, guildID, command)
		if err != nil {
			log.Fatalln("failed to create guild command:", err)
		}
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
