package discord

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Timestamp has a valid zero-value, which can be checked using the Valid()
// method. This is useful for optional timestamps such as EditedTimestamp.
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

// UnmarshalJSON parses a nullable RFC3339 string into time.
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

// MarshalJSON returns null if Timestamp is not valid (zero). It returns the
// time formatted in RFC3339 otherwise.
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

func TimeToMilliseconds(t time.Time) UnixMsTimestamp {
	return UnixMsTimestamp(t.UnixNano() / int64(time.Millisecond))
}

func (t UnixMsTimestamp) String() string {
	return t.Time().String()
}

func (t UnixMsTimestamp) Time() time.Time {
	return time.Unix(0, int64(t)*int64(time.Millisecond))
}

//

type Seconds int

// NullSecond is used in cases where null should be used instead of a number or
// omitted. This is similar to NullSnowflake.
const NullSecond = -1

func DurationToSeconds(dura time.Duration) Seconds {
	return Seconds(dura.Seconds())
}

func (s Seconds) MarshalJSON() ([]byte, error) {
	if s < 1 {
		return []byte("null"), nil
	} else {
		return []byte(strconv.Itoa(int(s))), nil
	}
}

func (s Seconds) String() string {
	return s.Duration().String()
}

func (s Seconds) Duration() time.Duration {
	return time.Duration(s) * time.Second
}

//

// Milliseconds is in float64 because some Discord events return time with a
// trailing decimal.
type Milliseconds float64

func DurationToMilliseconds(dura time.Duration) Milliseconds {
	return Milliseconds(dura.Milliseconds())
}

func (ms Milliseconds) String() string {
	return ms.Duration().String()
}

func (ms Milliseconds) Duration() time.Duration {
	const f64ms = Milliseconds(time.Millisecond)
	return time.Duration(ms * f64ms)
}
