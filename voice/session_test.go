package voice

import (
	"context"
	"log"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/internal/testenv"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/diamondburned/arikawa/v3/voice/testdata"
	"github.com/diamondburned/arikawa/v3/voice/voicegateway"
	"github.com/pkg/errors"
)

func TestIntegration(t *testing.T) {
	config := testenv.Must(t)

	wsutil.WSDebug = func(v ...interface{}) {
		_, file, line, _ := runtime.Caller(1)
		caller := file + ":" + strconv.Itoa(line)
		log.Println(append([]interface{}{caller}, v...)...)
	}

	s, err := state.New("Bot " + config.BotToken)
	if err != nil {
		t.Fatal("Failed to create a new state:", err)
	}
	AddIntents(s.Gateway)

	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.Open(ctx); err != nil {
			t.Fatal("Failed to connect:", err)
		}
	}()

	t.Cleanup(func() { s.Close() })

	v, err := NewSession(s)
	if err != nil {
		t.Fatal("Failed to create a new voice session:", err)
	}
	v.ErrorLog = func(err error) { t.Error(err) }

	// Grab a timer to benchmark things.
	finish := timer()

	// Add handler to receive speaking update beforehand.
	v.AddHandler(func(e *voicegateway.SpeakingEvent) {
		finish("receiving voice speaking event")
	})

	if err := v.JoinChannel(config.VoiceChID, false, false); err != nil {
		t.Fatal("failed to join a voice channel:", err)
	}

	t.Cleanup(func() {
		if err := v.Leave(); err != nil {
			t.Error("failed to leave voice channel gracefully:", err)
		}
	})

	finish("joining the voice channel")

	// Trigger speaking.
	if err := v.Speaking(voicegateway.Microphone); err != nil {
		t.Fatal("failed to start speaking:", err)
	}

	finish("sending the speaking command")

	// Create a context and only cancel it AFTER we're done sending silence
	// frames.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	doneCh := make(chan struct{})
	go func() {
		defer func() { doneCh <- struct{}{} }()
		if err := testdata.WriteOpus(v, "testdata/nico.dca"); err != nil {
			t.Error(err)
		}
	}()

	select {
	case <-ctx.Done():
		v.Leave()
	case <-doneCh:
		finish("copying the audio")
	}
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

// simple shitty benchmark thing
func timer() func(finished string) {
	var then = time.Now()

	return func(finished string) {
		now := time.Now()
		log.Println("Finished", finished+", took", now.Sub(then))
		then = now
	}
}
