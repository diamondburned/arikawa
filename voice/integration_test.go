// +build integration

package voice

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/diamondburned/arikawa/v2/voice/voicegateway"
)

func TestIntegration(t *testing.T) {
	config := mustConfig(t)

	wsutil.WSDebug = func(v ...interface{}) {
		_, file, line, _ := runtime.Caller(1)
		caller := file + ":" + strconv.Itoa(line)
		log.Println(append([]interface{}{caller}, v...)...)
	}

	v, err := NewVoiceFromToken("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("Failed to create a new voice session:", err)
	}
	v.Gateway.AddIntent(gateway.IntentGuildVoiceStates)

	v.ErrorLog = func(err error) {
		t.Error(err)
	}

	if err := v.Open(); err != nil {
		t.Fatal("Failed to connect:", err)
	}
	defer v.Close()

	// Validate the given voice channel.
	c, err := v.Channel(config.VoiceChID)
	if err != nil {
		t.Fatal("Failed to get channel:", err)
	}
	if c.Type != discord.GuildVoice {
		t.Fatal("Channel isn't a guild voice channel.")
	}

	log.Println("The voice channel's name is", c.Name)

	// Grab a timer to benchmark things.
	finish := timer()

	// Join the voice channel.
	vs, err := v.JoinChannel(c.GuildID, c.ID, false, false)
	if err != nil {
		t.Fatal("Failed to join channel:", err)
	}
	defer func() {
		log.Println("Disconnecting from the voice channel.")
		if err := vs.Disconnect(); err != nil {
			t.Fatal("Failed to disconnect:", err)
		}
	}()

	finish("joining the voice channel")

	// Trigger speaking.
	if err := vs.Speaking(voicegateway.Microphone); err != nil {
		t.Fatal("Failed to start speaking:", err)
	}
	defer func() {
		log.Println("Stopping speaking.") // sounds grammatically wrong
		if err := vs.StopSpeaking(); err != nil {
			t.Fatal("Failed to stop speaking:", err)
		}
	}()

	finish("sending the speaking command")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := vs.UseContext(ctx); err != nil {
		t.Fatal("failed to set ctx into vs:", err)
	}

	// Copy the audio?
	nicoReadTo(t, vs)

	finish("copying the audio")
}

type testConfig struct {
	BotToken  string
	VoiceChID discord.ChannelID
}

func mustConfig(t *testing.T) testConfig {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Fatal("Missing $BOT_TOKEN")
	}

	var sid = os.Getenv("VOICE_ID")
	if sid == "" {
		t.Fatal("Missing $VOICE_ID")
	}

	id, err := discord.ParseSnowflake(sid)
	if err != nil {
		t.Fatal("Invalid $VOICE_ID:", err)
	}

	return testConfig{
		BotToken:  token,
		VoiceChID: discord.ChannelID(id),
	}
}

// file is only a few bytes lolmao
func nicoReadTo(t *testing.T, dst io.Writer) {
	t.Helper()

	f, err := os.Open("testdata/nico.dca")
	if err != nil {
		t.Fatal("Failed to open nico.dca:", err)
	}
	defer f.Close()

	var lenbuf [4]byte

	for {
		if _, err := io.ReadFull(f, lenbuf[:]); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("failed to read:", err)
		}

		// Read the integer
		framelen := int64(binary.LittleEndian.Uint32(lenbuf[:]))

		// Copy the frame.
		if _, err := io.CopyN(dst, f, framelen); err != nil && err != io.EOF {
			t.Fatal("failed to write:", err)
		}
	}
}

// simple shitty benchmark thing
func timer() func(finished string) {
	var then = time.Now()

	return func(finished string) {
		now := time.Now()
		log.Println("Finished", finished+", took", now.Sub(then))
		then = now
	}
}
