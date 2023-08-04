package main

import (
	"context"
	"log"
	"os"

	"libdb.so/arikawa/v4/api"
	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/session"
	"libdb.so/arikawa/v4/utils/json/option"
)

// To run, do `GUILD_ID="GUILD ID" BOT_TOKEN="TOKEN HERE" go run .`

func main() {
	guildID := discord.GuildID(mustSnowflakeEnv("GUILD_ID"))

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("no $BOT_TOKEN given")
	}

	s := session.New("Bot " + token)

	app, err := s.CurrentApplication()
	if err != nil {
		log.Fatalln("failed to get application ID:", err)
	}

	s.AddHandler(func(e *gateway.InteractionCreateEvent) {
		var resp api.InteractionResponse

		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			if data.Name != "buttons" {
				resp = api.InteractionResponse{
					Type: api.MessageInteractionWithSource,
					Data: &api.InteractionResponseData{
						Content: option.NewNullableString("Unknown command: " + data.Name),
					},
				}
				break
			}
			// Send a message with a button back on slash commands.
			resp = api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("This is a message with a button!"),
					Components: discord.ComponentsPtr(
						&discord.ActionRowComponent{
							&discord.ButtonComponent{
								Label:    "Hello World!",
								CustomID: "first_button",
								Emoji:    &discord.ComponentEmoji{Name: "👋"},
								Style:    discord.PrimaryButtonStyle(),
							},
							&discord.ButtonComponent{
								Label:    "Secondary",
								CustomID: "second_button",
								Style:    discord.SecondaryButtonStyle(),
							},
							&discord.ButtonComponent{
								Label:    "Success",
								CustomID: "success_button",
								Style:    discord.SuccessButtonStyle(),
							},
							&discord.ButtonComponent{
								Label:    "Danger",
								CustomID: "danger_button",
								Style:    discord.DangerButtonStyle(),
							},
						},
						// This is automatically put into its own row.
						&discord.ButtonComponent{
							Label: "Link",
							Style: discord.LinkButtonStyle("https://google.com"),
						},
					),
				},
			}
		case discord.ComponentInteraction:
			resp = api.InteractionResponse{
				Type: api.UpdateMessage,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("Custom ID: " + string(data.ID())),
				},
			}
		default:
			log.Printf("unknown interaction type %T", e.Data)
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
			Name:        "buttons",
			Description: "Send an interactable message.",
		},
	}

	log.Println("Creating guild commands...")

	if _, err := s.BulkOverwriteGuildCommands(app.ID, guildID, newCommands); err != nil {
		log.Fatalln("failed to create guild command:", err)
	}

	log.Println("Guild commands created. Bot is ready.")

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
