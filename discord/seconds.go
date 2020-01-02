package discord

import "time"

type Seconds uint

func DurationToSeconds(dura time.Duration) Seconds {
	return Seconds(dura.Seconds())
}

func (s Seconds) String() string {
	return s.Duration().String()
}

func (s Seconds) Duration() time.Duration {
	return time.Duration(s) * time.Second
}
