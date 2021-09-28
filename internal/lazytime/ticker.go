package lazytime

import "time"

type Ticker struct {
	C <-chan time.Time

	ticker *time.Ticker
}

// Reset resets the ticker. If this is the first time calling, then a new timer
// is created.
func (t *Ticker) Reset(d time.Duration) {
	if t.ticker == nil {
		t.ticker = time.NewTicker(d)
		t.C = t.ticker.C
	} else {
		t.ticker.Reset(d)
	}
}

// Stop stops the ticker. If the ticker has never been used, then it does
// nothing.
func (t *Ticker) Stop() {
	if t.ticker == nil {
		return
	}

	t.ticker.Stop()
}
