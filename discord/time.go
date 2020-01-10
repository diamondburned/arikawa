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

//

type UnixTimestamp int64

func (t UnixTimestamp) String() string {
	return t.Time().String()
}

func (t UnixTimestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

//

type UnixMsTimestamp int64

func (t UnixMsTimestamp) String() string {
	return t.Time().String()
}

func (t UnixMsTimestamp) Time() time.Time {
	return time.Unix(0, int64(t)*int64(time.Millisecond))
}

//

type Seconds int

func DurationToSeconds(dura time.Duration) Seconds {
	return Seconds(dura.Seconds())
}

func (s Seconds) String() string {
	return s.Duration().String()
}

func (s Seconds) Duration() time.Duration {
	return time.Duration(s) * time.Second
}

//

type Milliseconds int

func DurationToMilliseconds(dura time.Duration) Milliseconds {
	return Milliseconds(dura.Milliseconds())
}

func (ms Milliseconds) String() string {
	return ms.Duration().String()
}

func (ms Milliseconds) Duration() time.Duration {
	return time.Duration(ms) * time.Millisecond
}
