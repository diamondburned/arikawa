package discord

import (
	"strconv"
	"strings"
	"time"
)

const DiscordEpoch = 1420070400000 * int64(time.Millisecond)

type Snowflake int64

func NewSnowflake(t time.Time) Snowflake {
	return Snowflake(TimeToDiscordEpoch(t) << 22)
}

const Me = Snowflake(-1)

func (s *Snowflake) UnmarshalJSON(v []byte) error {
	id := strings.Trim(string(v), `"`)
	if id == "null" {
		return nil
	}

	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(i)
	return nil
}

func (s *Snowflake) MarshalJSON() ([]byte, error) {
	var id string

	switch i := int64(*s); i {
	case -1: // @me
		id = "@me"
	case 0:
		return []byte("null"), nil
	default:
		id = strconv.FormatInt(i, 10)
	}

	return []byte(`"` + id + `"`), nil
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) Valid() bool {
	return uint64(s) > 0
}

func (s Snowflake) Time() time.Time {
	return time.Unix(0, int64(s)>>22*1000000+DiscordEpoch)
}

func (s Snowflake) Worker() uint8 {
	return uint8(s & 0x3E0000)
}

func (s Snowflake) PID() uint8 {
	return uint8(s & 0x1F000 >> 12)
}

func (s Snowflake) Increment() uint16 {
	return uint16(s & 0xFFF)
}

func TimeToDiscordEpoch(t time.Time) int64 {
	return t.UnixNano()/int64(time.Millisecond) - DiscordEpoch
}
