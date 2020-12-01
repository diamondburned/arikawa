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
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/wsutil"
	"github.com/diamondburned/arikawa/v2/voice/voicegateway"
	"github.com/pkg/errors"
)

func TestIntegration(t *testing.T) {
	config := mustConfig(t)

	wsutil.WSDebug = func(v ...interface{}) {
		_, file, line, _ := runtime.Caller(1)
		caller := file + ":" + strconv.Itoa(line)
		log.Println(append([]interface{}{caller}, v...)...)
	}

	v, err := NewFromToken("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("Failed to create a new voice session:", err)
	}
	v.Gateway.AddIntents(gateway.IntentGuildVoiceStates)

	v.ErrorLog = func(err error) {
		t.Error(err)
	}

	if err := v.Open(); err != nil {
		t.Fatal("Failed to connect:", err)
	}
	t.Cleanup(func() { v.Close() })

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

	// Join the voice channel concurrently.
	raceValue := raceMe(t, "failed to join voice channel", func() (interface{}, error) {
		return v.JoinChannel(c.ID, false, false)
	})
	vs := raceValue.(*Session)

	t.Cleanup(func() {
		log.Println("Disconnecting from the voice channel concurrently.")

		raceMe(t, "failed to disconnect", func() (interface{}, error) {
			return nil, vs.Disconnect()
		})
	})

	finish("joining the voice channel")

	// Add handler to receive speaking update
	vs.AddHandler(func(e *voicegateway.SpeakingEvent) {
		finish("received voice speaking event")
	})

	// Create a context and only cancel it AFTER we're done sending silence
	// frames.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	// Trigger speaking.
	if err := vs.Speaking(voicegateway.Microphone); err != nil {
		t.Fatal("failed to start speaking:", err)
	}

	finish("sending the speaking command")

	if err := vs.UseContext(ctx); err != nil {
		t.Fatal("failed to set ctx into vs:", err)
	}

	f, err := os.Open("testdata/nico.dca")
	if err != nil {
		t.Fatal("Failed to open nico.dca:", err)
	}
	defer f.Close()

	var lenbuf [4]byte

	// Copy the audio?
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
		if _, err := io.CopyN(vs, f, framelen); err != nil && err != io.EOF {
			t.Fatal("failed to write:", err)
		}
	}

	finish("copying the audio")
}

// raceMe intentionally calls fn multiple times in goroutines to ensure it's not
// racy.
func raceMe(t *testing.T, wrapErr string, fn func() (interface{}, error)) interface{} {
	const n = 3 // run 3 times
	t.Helper()

	// It is very ironic how this method itself is racy.

	var wgr sync.WaitGroup
	var mut sync.Mutex
	var val interface{}
	var err error

	for i := 0; i < n; i++ {
		wgr.Add(1)
		go func() {
			v, e := fn()

			mut.Lock()
			val = v
			err = e
			mut.Unlock()

			if e != nil {
				log.Println("Potential race test error:", e)
			}

			wgr.Done()
		}()
	}

	wgr.Wait()

	if err != nil {
		t.Fatal("Race test failed:", errors.Wrap(err, wrapErr))
	}

	return val
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
