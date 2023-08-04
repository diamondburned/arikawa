package main

import (
	"context"
	"log"
	"os"

	"libdb.so/arikawa/v4/api"
	"libdb.so/arikawa/v4/api/cmdroute"
	"libdb.so/arikawa/v4/gateway"
	"libdb.so/arikawa/v4/state"
	"libdb.so/arikawa/v4/utils/json/option"
)

var commands = []api.CreateCommandData{{Name: "ping", Description: "Ping!"}}

func main() {
	r := cmdroute.NewRouter()
	r.AddFunc("ping", func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
		return &api.InteractionResponseData{Content: option.NewNullableString("Pong!")}
	})

	s := state.New("Bot " + os.Getenv("BOT_TOKEN"))
	s.AddInteractionHandler(r)
	s.AddIntents(gateway.IntentGuilds)

	if err := cmdroute.OverwriteCommands(s, commands); err != nil {
		log.Fatalln("cannot update commands:", err)
	}

	if err := s.Connect(context.TODO()); err != nil {
		log.Println("cannot connect:", err)
	}
}
