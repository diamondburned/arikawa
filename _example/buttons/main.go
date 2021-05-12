package main

import (
	"log"
	"os"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
)

// To run, do `APP_ID="APP ID" GUILD_ID="GUILD ID" BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	appID := discord.AppID(mustSnowflakeEnv("APP_ID"))
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

	s.AddHandler(func(e *gateway.InteractionCreateEvent) {
		if e.Type == gateway.CommandInteraction {
			// Send a message with a button back on slash commands.
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: "This is a message with a button!",
					Components: []discord.Component{
						discord.ActionRow{
							Components: []discord.Component{
								discord.Button{
									Label:    "Hello World!",
									CustomID: "first_button",
									Emoji: &discord.ButtonEmoji{
										Name: "👋",
									},
									Style: discord.PrimaryButton,
								},
								discord.Button{
									Label:    "Secondary",
									CustomID: "second_button",
									Style:    discord.SecondaryButton,
								},
								discord.Button{
									Label:    "Success",
									CustomID: "success_button",
									Style:    discord.SuccessButton,
								},
								discord.Button{
									Label:    "Danger",
									CustomID: "danger_button",
									Style:    discord.DangerButton,
								},
								discord.Button{
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

		if e.Type != gateway.ButtonInteraction {
			return
		}

		data := api.InteractionResponse{
			Type: api.UpdateMessage,
			Data: &api.InteractionResponseData{
				Content: "Custom ID: " + e.Data.CustomID,
			},
		}

		if err := s.RespondInteraction(e.ID, e.Token, data); err != nil {
			log.Println("failed to send interaction callback:", err)
		}
	})

	s.Gateway.AddIntents(gateway.IntentGuilds)
	s.Gateway.AddIntents(gateway.IntentGuildMessages)

	if err := s.Open(); err != nil {
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
