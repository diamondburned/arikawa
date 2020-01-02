package discord

import (
	"encoding/json"
	"time"
)

type Timestamp time.Time

const TimestampFormat = time.RFC3339 // same as ISO8601

var (
	_ json.Unmarshaler = (*Timestamp)(nil)
	_ json.Marshaler   = (*Timestamp)(nil)
)

func (t *Timestamp) UnmarshalJSON(v []byte) error {
	r, err := time.Parse(TimestampFormat, string(v))
	if err != nil {
		return err
	}

	*t = Timestamp(r)
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(TimestampFormat) + `"`), nil
}

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
