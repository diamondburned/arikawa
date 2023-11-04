package voice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/arikawa/v3/voice/testdata"
	"github.com/diamondburned/arikawa/v3/voice/udp"
	"github.com/diamondburned/arikawa/v3/voice/voicegateway"
)

func TestMain(m *testing.M) {
	ws.WSDebug = func(v ...interface{}) {
		_, file, line, _ := runtime.Caller(1)
		caller := file + ":" + strconv.Itoa(line)
		log.Println(append([]interface{}{caller}, v...)...)
	}

	code := m.Run()
	os.Exit(code)
}

type testState struct {
	*state.State
	channel *discord.Channel
}

func testOpen(t *testing.T) *testState {
	config := testenv.Must(t)

	s := state.New("Bot " + config.BotToken)
	AddIntents(s)

	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		if err := s.Open(ctx); err != nil {
			t.Fatal("failed to connect:", err)
		}
	}()

	t.Cleanup(func() { s.Close() })

	// Validate the given voice channel.
	c, err := s.Channel(config.VoiceChID)
	if err != nil {
		t.Fatal("failed to get channel:", err)
	}
	if c.Type != discord.GuildVoice {
		t.Fatal("channel isn't a guild voice channel.")
	}

	t.Log("The voice channel's name is", c.Name)

	return &testState{
		State:   s,
		channel: c,
	}
}

func TestIntegration(t *testing.T) {
	state := testOpen(t)

	t.Run("1st", func(t *testing.T) { testIntegrationOnce(t, state) })
	t.Run("2nd", func(t *testing.T) { testIntegrationOnce(t, state) })
}

func testIntegrationOnce(t *testing.T, s *testState) {
	v, err := NewSession(s)
	if err != nil {
		t.Fatal("failed to create a new voice session:", err)
	}

	// Grab a timer to benchmark things.
	finish := timer()

	// Add handler to receive speaking update beforehand.
	v.AddHandler(func(e *voicegateway.SpeakingEvent) {
		finish("receiving voice speaking event")
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	if err := v.JoinChannelAndSpeak(ctx, s.channel.ID, false, false); err != nil {
		t.Fatal("failed to join voice:", err)
	}

	t.Cleanup(func() {
		t.Log("Leaving the voice channel concurrently.")

		raceMe(t, "failed to leave voice channel", func() error {
			return v.Leave(ctx)
		})
	})

	finish("joining the voice channel")

	t.Cleanup(func() {})

	finish("sending the speaking command")

	doneCh := make(chan struct{})
	go func() {
		if err := testdata.WriteOpus(v, testdata.Nico); err != nil {
			t.Error(err)
		}
		doneCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		t.Error("timed out waiting for voice to be done")
	case <-doneCh:
		finish("copying the audio")
	}
}

// raceMe intentionally calls fn multiple times in goroutines to ensure it's not
// racy.
func raceMe(t *testing.T, wrapErr string, fn func() error) {
	const n = 3 // run 3 times
	t.Helper()

	// It is very ironic how this method itself is racy.

	var wgr sync.WaitGroup
	var mut sync.Mutex
	var err error

	for i := 0; i < n; i++ {
		wgr.Add(1)
		go func() {
			e := fn()

			mut.Lock()
			if e != nil {
				err = e
			}
			mut.Unlock()

			if e != nil {
				t.Log("Potential race test error:", e)
			}

			wgr.Done()
		}()
	}

	wgr.Wait()

	if err != nil {
		t.Fatal("race test failed:", fmt.Errorf("%s: %w", wrapErr, err))
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

func TestKickedOut(t *testing.T) {
	err := testReconnect(t, func(s *testState) {
		me, err := s.Me()
		if err != nil {
			t.Fatal("cannot get me")
		}

		if err := s.ModifyMember(s.channel.GuildID, me.ID, api.ModifyMemberData{
			// Kick the bot out.
			VoiceChannel: discord.NullChannelID,
		}); err != nil {
			t.Error("cannot kick the bot out:", err)
		}
	})

	if !errors.Is(err, udp.ErrManagerClosed) {
		t.Error("unexpected error while sending nico.dca:", err)
	}
}

func TestRegionChange(t *testing.T) {
	var state *testState
	err := testReconnect(t, func(s *testState) {
		state = s
		t.Log("got voice region", s.channel.RTCRegionID)

		regions, err := s.VoiceRegionsGuild(s.channel.GuildID)
		if err != nil {
			t.Error("cannot get voice region:", err)
			return
		}

		rand.Shuffle(len(regions), func(i, j int) {
			regions[i], regions[j] = regions[j], regions[i]
		})

		var anyRegion string
		for _, region := range regions {
			if region.ID != s.channel.RTCRegionID {
				anyRegion = region.ID
				break
			}
		}

		t.Log("changing voice region to", anyRegion)

		if err := s.ModifyChannel(s.channel.ID, api.ModifyChannelData{
			RTCRegionID: option.NewNullableString(anyRegion),
		}); err != nil {
			t.Error("cannot change voice region:", err)
		}
	})

	if err != nil {
		t.Error("unexpected error while sending nico.dca:", err)
	}

	s := state

	// Change voice region back.
	if err := s.ModifyChannel(s.channel.ID, api.ModifyChannelData{
		RTCRegionID: option.NewNullableString(s.channel.RTCRegionID),
	}); err != nil {
		t.Error("cannot change voice region back:", err)
	}

	t.Log("changed voice region back to", s.channel.RTCRegionID)
}

func testReconnect(t *testing.T, interrupt func(*testState)) error {
	s := testOpen(t)

	v, err := NewSession(s)
	if err != nil {
		t.Fatal("cannot")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	if err := v.JoinChannelAndSpeak(ctx, s.channel.ID, false, false); err != nil {
		t.Fatal("failed to join voice:", err)
	}

	t.Cleanup(func() {
		if err := v.Speaking(ctx, voicegateway.NotSpeaking); err != nil {
			t.Error("cannot stop speaking:", err)
		}
		if err := v.Leave(ctx); err != nil {
			t.Error("cannot leave voice:", err)
		}
	})

	// Ensure the channel is buffered so we can send into it. Write may not be
	// called often enough to immediately receive a tick from the unbuffered
	// timer.
	oneSec := make(chan struct{}, 1)
	go func() {
		<-time.After(450 * time.Millisecond)
		oneSec <- struct{}{}
	}()

	// Use a WriterFunc so we can interrupt the writing.
	// Give 1s for the function to write before interrupting it; we already know
	// that the saved dca file is longer than 1s, so we're fine doing this.
	interruptWriter := testdata.WriterFunc(func(b []byte) (int, error) {
		select {
		case <-oneSec:
			interrupt(s)
		default:
			// ok
		}

		return v.Write(b)
	})

	return testdata.WriteOpus(interruptWriter, testdata.Nico)
}
