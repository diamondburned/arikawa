package ws

import (
	"time"

	"golang.org/x/time/rate"
)

// SendBurst determines the number of gateway commands that can be sent all at
// once before being throttled. The higher the burst, the slower the rate
// limiter recovers.
var SendBurst = 5

// NewSendLimiter returns a rate limiter for throttling gateway commands.
func NewSendLimiter() *rate.Limiter {
	const perMinute = 120
	return rate.NewLimiter(
		// Permit r = minute / (120 - b) commands per second.
		rate.Every(time.Minute/(perMinute-time.Duration(SendBurst))),
		SendBurst,
	)
}

// NewDialLimiter returns a rate limiter for throttling new gateway connections.
func NewDialLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(5*time.Second), 1)
}

// NewIdentityLimiter returns a rate limiter for throttling gateway Identify
// commands.
func NewIdentityLimiter() *rate.Limiter {
	return NewDialLimiter() // same
}

// NewGlobalIdentityLimiter returns a rate limiter for throttling global
// gateway Identify commands.
func NewGlobalIdentityLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(24*time.Hour), 1000)
}
