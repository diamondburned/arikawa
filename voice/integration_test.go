// +build integration

package voice

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"testing"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
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
func nicoReader(t *testing.T) (read func() []byte) {
	f, err := os.Open("testdata/nico.dca")
	if err != nil {
		t.Fatal("Failed to open nico.dca:", err)
	}

	t.Cleanup(func() {
		f.Close()
	})

	var lenbuf [4]byte

	return func() []byte {
		if _, err := io.ReadFull(f, lenbuf[:]); !catchRead(t, err) {
			return nil
		}

		// Read the integer
		framelen := int(binary.LittleEndian.Uint32(lenbuf[:]))

		// Read exactly frame
		frame := make([]byte, framelen)

		if _, err := io.ReadFull(f, frame); !catchRead(t, err) {
			return nil
		}

		return frame
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

	WSDebug = func(v ...interface{}) {
		log.Println(v...)
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

	// Validate the given voice channel.
	c, err := s.Channel(config.VoiceChID)
	if err != nil {
		t.Fatal("Failed to get channel:", err)
	}
	if c.Type != discord.GuildVoice {
		t.Fatal("Channel isn't a guild voice channel.")
	}

	conn, err := v.JoinChannel(c.GuildID, c.ID, false, false)
	if err != nil {
		t.Fatal("Failed to join channel:", err)
	}

	// Grab the file in the local test data.
	read := nicoReader(t)

	// Trigger speaking.
	if err := conn.Speaking(Microphone); err != nil {
		t.Fatal("Failed to start speaking:", err)
	}

	// Copy the audio?
	for bytes := read(); bytes != nil; bytes = read() {
		conn.OpusSend <- bytes
		// conn.Write(bytes)
	}

	// Finish speaking.
	if err := conn.StopSpeaking(); err != nil {
		t.Fatal("Failed to stop speaking:", err)
	}

	if err := conn.Disconnect(s.Gateway); err != nil {
		t.Fatal("Failed to disconnect:", err)
	}
}
