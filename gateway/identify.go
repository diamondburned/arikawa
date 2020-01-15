package gateway

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

type Identifier struct {
	IdentifyData

	IdentifyShortLimit  *rate.Limiter `json:"-"`
	IdentifyGlobalLimit *rate.Limiter `json:"-"`
}

func DefaultIdentifier(token string) *Identifier {
	return NewIdentifier(IdentifyData{
		Token:      token,
		Properties: Identity,
		Shard:      DefaultShard(),

		Compress:          true,
		LargeThreshold:    50,
		GuildSubscription: true,
	})
}

func NewIdentifier(data IdentifyData) *Identifier {
	return &Identifier{
		IdentifyData:        data,
		IdentifyShortLimit:  rate.NewLimiter(rate.Every(5*time.Second), 1),
		IdentifyGlobalLimit: rate.NewLimiter(rate.Every(24*time.Hour), 1000),
	}
}

func (i *Identifier) Wait(ctx context.Context) error {
	if err := i.IdentifyShortLimit.Wait(ctx); err != nil {
		return errors.Wrap(err, "Can't wait for short limit")
	}
	if err := i.IdentifyGlobalLimit.Wait(ctx); err != nil {
		return errors.Wrap(err, "Can't wait for global limit")
	}
	return nil
}
