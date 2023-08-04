package voice_test

import (
	"context"
	"log"
	"os"
	"os/signal"
	"testing"

	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/internal/testenv"
	"libdb.so/arikawa/v4/state"
	"libdb.so/arikawa/v4/voice"
	"libdb.so/arikawa/v4/voice/testdata"
)

var (
	token     string
	channelID discord.ChannelID
)

func init() {
	e, err := testenv.GetEnv()
	if err == nil {
		token = e.BotToken
		channelID = e.VoiceChID
	}
}

// make godoc not show the full file
func TestNoop(t *testing.T) {
	t.Skip("noop")
}

func ExampleSession() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	s := state.New("Bot " + token)

	// This is required for bots.
	voice.AddIntents(s)

	if err := s.Open(ctx); err != nil {
		log.Fatalln("failed to open gateway:", err)
	}
	defer s.Close()

	v, err := voice.NewSession(s)
	if err != nil {
		log.Fatalln("failed to create voice session:", err)
	}

	if err := v.JoinChannelAndSpeak(ctx, channelID, false, false); err != nil {
		log.Fatalln("failed to join voice channel:", err)
	}
	defer v.Leave(ctx)

	if err := testdata.WriteOpus(v, "testdata/nico.dca"); err != nil {
		log.Fatalln("failed to write opus:", err)
	}
}
