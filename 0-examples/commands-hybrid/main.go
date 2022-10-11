package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/pkg/errors"
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

	var h handler

	var (
		webhookAddr   = os.Getenv("WEBHOOK_ADDR")
		webhookPubkey = os.Getenv("WEBHOOK_PUBKEY")
	)

	if webhookAddr != "" {
		h.s = state.NewAPIOnlyState(token, nil)

		srv, err := webhook.NewInteractionServer(webhookPubkey, &h, true)
		if err != nil {
			log.Fatalln("cannot create interaction server:", err)
		}

		if err := overwriteCommands(h.s); err != nil {
			log.Fatalln("cannot update commands:", err)
		}

		log.Println("listening and serving at", webhookAddr+"/")
		log.Fatalln(http.ListenAndServe(webhookAddr, srv))
	} else {
		h.s = state.New("Bot " + token)
		h.s.AddInteractionHandler(&h)
		h.s.AddIntents(gateway.IntentGuilds)
		h.s.AddHandler(func(*gateway.ReadyEvent) {
			me, _ := h.s.Me()
			log.Println("connected to the gateway as", me.Tag())
		})

		if err := overwriteCommands(h.s); err != nil {
			log.Fatalln("cannot update commands:", err)
		}

		if err := h.s.Connect(context.Background()); err != nil {
			log.Fatalln("cannot connect:", err)
		}
	}
}

func overwriteCommands(s *state.State) error {
	app, err := s.CurrentApplication()
	if err != nil {
		return errors.Wrap(err, "cannot get current app ID")
	}

	_, err = s.BulkOverwriteCommands(app.ID, commands)
	return err
}

type handler struct {
	s *state.State
}

func (h *handler) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	switch data := ev.Data.(type) {
	case *discord.CommandInteraction:
		switch data.Name {
		case "ping":
			return h.cmdPing(ev, data)
		case "echo":
			return h.cmdEcho(ev, data)
		case "thonk":
			return h.cmdThonk(ev, data)
		default:
			return errorResponse(fmt.Errorf("unknown command %q", data.Name))
		}
	default:
		return errorResponse(fmt.Errorf("unknown interaction %T", ev.Data))
	}
}

func (h *handler) cmdPing(ev *discord.InteractionEvent, _ *discord.CommandInteraction) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString("Pong!"),
		},
	}
}

func (h *handler) cmdEcho(ev *discord.InteractionEvent, data *discord.CommandInteraction) *api.InteractionResponse {
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
			AllowedMentions: &api.AllowedMentions{},
		},
	}
}

func (h *handler) cmdThonk(ev *discord.InteractionEvent, data *discord.CommandInteraction) *api.InteractionResponse {
	go func() {
		time.Sleep(time.Duration(3+rand.Intn(5)) * time.Second)

		h.s.FollowUpInteraction(ev.AppID, ev.Token, api.InteractionResponseData{
			Content: option.NewNullableString("https://tenor.com/view/thonk-thinking-sun-thonk-sun-thinking-sun-gif-14999983"),
		})
	}()

	return deferResponse(0)
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

func deferResponse(flags discord.MessageFlags) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.DeferredMessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: flags,
		},
	}
}
