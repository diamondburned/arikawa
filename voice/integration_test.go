// +build integration

package voice

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/diamondburned/arikawa/voice/voicegateway"
)

type testConfig struct {
	BotToken  string
	VoiceChID discord.Snowflake
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
		VoiceChID: id,
	}
}

// file is only a few bytes lolmao
func nicoReadTo(t *testing.T, dst io.Writer) {
	f, err := os.Open("testdata/nico.dca")
	if err != nil {
		t.Fatal("Failed to open nico.dca:", err)
	}

	t.Cleanup(func() {
		f.Close()
	})

	var lenbuf [4]byte

	for {
		if _, err := io.ReadFull(f, lenbuf[:]); !catchRead(t, err) {
			return
		}

		// Read the integer
		framelen := int64(binary.LittleEndian.Uint32(lenbuf[:]))

		// Copy the frame.
		if _, err := io.CopyN(dst, f, framelen); !catchRead(t, err) {
			return
		}
	}
}

func catchRead(t *testing.T, err error) bool {
	t.Helper()

	if err == io.EOF {
		return false
	}
	if err != nil {
		t.Fatal("Failed to read:", err)
	}
	return true
}

func TestIntegration(t *testing.T) {
	config := mustConfig(t)

	wsutil.WSDebug = func(v ...interface{}) {
		_, file, line, _ := runtime.Caller(1)
		caller := file + ":" + strconv.Itoa(line)
		log.Println(append([]interface{}{caller}, v...)...)
	}

	// heart.Debug = func(v ...interface{}) {
	// 	log.Println(append([]interface{}{"Pacemaker:"}, v...)...)
	// }

	s, err := state.New("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("Failed to create a new session:", err)
	}

	v := NewVoice(s)

	if err := s.Open(); err != nil {
		t.Fatal("Failed to connect:", err)
	}
	defer s.Close()

	// Validate the given voice channel.
	c, err := s.Channel(config.VoiceChID)
	if err != nil {
		t.Fatal("Failed to get channel:", err)
	}
	if c.Type != discord.GuildVoice {
		t.Fatal("Channel isn't a guild voice channel.")
	}

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

	// Copy the audio?
	nicoReadTo(t, vs)

	finish("copying the audio")
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
