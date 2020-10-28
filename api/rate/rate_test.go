package rate

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// https://github.com/bwmarrin/discordgo/blob/master/ratelimit_test.go

func mockRequest(t *testing.T, l *Limiter, path string, headers http.Header) {
	if err := l.Acquire(context.Background(), path); err != nil {
		t.Fatal("Failed to acquire lock:", err)
	}

	if err := l.Release(path, headers); err != nil {
		t.Fatal("Failed to release lock:", err)
	}
}

// This test takes ~2 seconds to run
func TestRatelimitReset(t *testing.T) {
	l := NewLimiter("")

	const msToSec = time.Second / time.Millisecond

	until := time.Now().Add(2 * time.Second)
	reset := float64(until.UnixNano()/int64(time.Millisecond)) / float64(msToSec)

	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "0")
	headers.Set("X-RateLimit-Reset", fmt.Sprintf("%.3f", reset))
	headers.Set("Date", time.Now().Format(time.RFC850))

	sent := time.Now()
	mockRequest(t, l, "/guilds/99/channels", headers)
	mockRequest(t, l, "/guilds/55/channels", headers)
	mockRequest(t, l, "/guilds/66/channels", headers)

	// call it again
	mockRequest(t, l, "/guilds/99/channels", headers)
	mockRequest(t, l, "/guilds/55/channels", headers)
	mockRequest(t, l, "/guilds/66/channels", headers)

	// We hit the same endpoint 2 times, so we should only be ratelimited 2
	// second and always less than 4 seconds (unless you're on a stoneage
	// computer or using swap or something...)
	if since := time.Since(sent); since >= time.Second && since < time.Second*4 {
		t.Log("OK", since)
	} else {
		t.Error("did not ratelimit correctly, got:", since)
	}
}

// This test takes ~1 seconds to run
func TestRatelimitGlobal(t *testing.T) {
	l := NewLimiter("")

	headers := http.Header{}
	headers.Set("X-RateLimit-Global", "1.002")
	// Reset for approx 1 seconds from now
	headers.Set("Retry-After", "1")

	sent := time.Now()

	// This should trigger a global ratelimit
	mockRequest(t, l, "/guilds/99/channels", headers)
	time.Sleep(time.Millisecond * 100)

	// This shouldn't go through in less than 1 second
	mockRequest(t, l, "/guilds/55/channels", headers)

	if time.Since(sent) >= time.Second && time.Since(sent) < time.Second*2 {
		t.Log("OK", time.Since(sent))
	} else {
		t.Error("did not ratelimit correctly, got:", time.Since(sent))
	}
}
