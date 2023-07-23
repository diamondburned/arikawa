package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var commands = []api.CreateCommandData{
	{
		Name:        "ping",
		Description: "ping pong!",
	},
	{
		Name:        "echo",
		Description: "echo back the argument",
		Options: []discord.CommandOption{
			&discord.StringOption{
				OptionName:  "argument",
				Description: "what's echoed back",
				Required:    true,
			},
		},
	},
	{
		Name:        "thonk",
		Description: "biiiig thonk",
	},
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	var (
		webhookAddr   = os.Getenv("WEBHOOK_ADDR")
		webhookPubkey = os.Getenv("WEBHOOK_PUBKEY")
	)

	if webhookAddr != "" {
		state := state.NewAPIOnlyState(token, nil)

		h := newHandler(state)

		if err := overwriteCommands(state); err != nil {
			log.Fatalln("cannot update commands:", err)
		}

		srv, err := webhook.NewInteractionServer(webhookPubkey, h)
		if err != nil {
			log.Fatalln("cannot create interaction server:", err)
		}

		log.Println("listening and serving at", webhookAddr+"/")
		log.Fatalln(http.ListenAndServe(webhookAddr, srv))
	} else {
		state := state.New("Bot " + token)
		state.AddIntents(gateway.IntentGuilds)
		state.AddHandler(func(*gateway.ReadyEvent) {
			me, _ := state.Me()
			log.Println("connected to the gateway as", me.Tag())
		})

		h := newHandler(state)
		state.AddInteractionHandler(h)

		if err := overwriteCommands(state); err != nil {
			log.Fatalln("cannot update commands:", err)
		}

		if err := h.s.Connect(context.Background()); err != nil {
			log.Fatalln("cannot connect:", err)
		}
	}
}

func overwriteCommands(s *state.State) error {
	return cmdroute.OverwriteCommands(s, commands)
}

type handler struct {
	*cmdroute.Router
	s *state.State
}

func newHandler(s *state.State) *handler {
	h := &handler{s: s}

	h.Router = cmdroute.NewRouter()
	// Automatically defer handles if they're slow.
	h.Use(cmdroute.Deferrable(s, cmdroute.DeferOpts{}))
	h.AddFunc("ping", h.cmdPing)
	h.AddFunc("echo", h.cmdEcho)
	h.AddFunc("thonk", h.cmdThonk)

	return h
}

func (h *handler) cmdPing(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString("Pong!"),
		},
	}
}

func (h *handler) cmdEcho(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponse {
	var options struct {
		Arg string `discord:"argument"`
	}

	if err := data.Options.Unmarshal(&options); err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content:         option.NewNullableString(options.Arg),
			AllowedMentions: &api.AllowedMentions{}, // don't mention anyone
		},
	}
}

func (h *handler) cmdThonk(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponse {
	time.Sleep(time.Duration(3+rand.Intn(5)) * time.Second)
	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString("https://tenor.com/view/thonk-thinking-sun-thonk-sun-thinking-sun-gif-14999983"),
		},
	}
}

func errorResponse(err error) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content:         option.NewNullableString("**Error:** " + err.Error()),
			Flags:           discord.EphemeralMessage,
			AllowedMentions: &api.AllowedMentions{ /* none */ },
		},
	}
}
