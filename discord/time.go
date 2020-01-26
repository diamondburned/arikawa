package discord

import (
	"encoding/json"
	"strings"
	"time"
)

type Timestamp time.Time

const TimestampFormat = time.RFC3339 // same as ISO8601

var (
	_ json.Unmarshaler = (*Timestamp)(nil)
	_ json.Marshaler   = (*Timestamp)(nil)
)

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp(t)
}

func NowTimestamp() Timestamp {
	return NewTimestamp(time.Now())
}

func (t *Timestamp) UnmarshalJSON(v []byte) error {
	str := strings.Trim(string(v), `"`)
	if str == "null" {
		return nil
	}

	r, err := time.Parse(TimestampFormat, str)
	if err != nil {
		return err
	}

	*t = Timestamp(r)
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if !t.Valid() {
		return []byte("null"), nil
	}

	return []byte(`"` + t.Format(TimestampFormat) + `"`), nil
}

func (t Timestamp) Valid() bool {
	return !t.Time().IsZero()
}

func (t Timestamp) Format(fmt string) string {
	return t.Time().Format(fmt)
}

func (t Timestamp) Time() time.Time {
	return time.Time(t)
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
