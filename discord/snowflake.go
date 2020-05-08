package discord

import (
	"strconv"
	"strings"
	"time"
)

// DiscordEpoch is the Discord epoch constant in time.Duration (nanoseconds)
// since Unix epoch.
const DiscordEpoch = 1420070400000 * time.Millisecond

// DurationSinceDiscordEpoch returns the duration from the Discord epoch to
// current.
func DurationSinceDiscordEpoch(t time.Time) time.Duration {
	return time.Duration(t.UnixNano()) - DiscordEpoch
}

type Snowflake int64

// NullSnowflake gets encoded into a null. This is used for
// optional and nullable snowflake fields.
const NullSnowflake Snowflake = -1

func NewSnowflake(t time.Time) Snowflake {
	return Snowflake((DurationSinceDiscordEpoch(t) / time.Millisecond) << 22)
}

func ParseSnowflake(sf string) (Snowflake, error) {
	i, err := strconv.ParseInt(sf, 10, 64)
	if err != nil {
		return 0, err
	}

	return Snowflake(i), nil
}

func (s *Snowflake) UnmarshalJSON(v []byte) error {
	id := strings.Trim(string(v), `"`)
	if id == "null" {
		// Use -1 for null.
		*s = -1
		return nil
	}

	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(i)
	return nil
}

func (s Snowflake) MarshalJSON() ([]byte, error) {
	if s < 1 {
		return []byte("null"), nil
	} else {
		return []byte(`"` + strconv.FormatInt(int64(s), 10) + `"`), nil
	}
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) Valid() bool {
	return uint64(s) > 0
}

func (s Snowflake) Time() time.Time {
	unixnano := ((time.Duration(s) >> 22) * time.Millisecond) + DiscordEpoch
	return time.Unix(0, int64(unixnano))
}

func (s Snowflake) Worker() uint8 {
	return uint8(s & 0x3E0000 >> 17)
}

func (s Snowflake) PID() uint8 {
	return uint8(s & 0x1F000 >> 12)
}

func (s Snowflake) Increment() uint16 {
	return uint16(s & 0xFFF)
}
