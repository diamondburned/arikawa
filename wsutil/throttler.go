package wsutil

import (
	"time"

	"golang.org/x/time/rate"
)

func NewSendLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Minute), 120)
}
