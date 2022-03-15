package discord

import (
	"strconv"
	"strings"
	"time"
)

// Epoch is the Discord epoch constant in time.Duration (nanoseconds)
// since Unix epoch.
const Epoch = 1420070400000 * time.Millisecond

// DurationSinceEpoch returns the duration from the Discord epoch to current.
func DurationSinceEpoch(t time.Time) time.Duration {
	return time.Duration(t.UnixNano()) - Epoch
}

//go:generate go run ../utils/cmd/gensnowflake -o snowflake_types.go AppID AttachmentID AuditLogEntryID ChannelID CommandID EmojiID GuildID IntegrationID InteractionID MessageID RoleID StageID StickerID StickerPackID TeamID UserID WebhookID

// Mention generates the mention syntax for this channel ID.
func (s ChannelID) Mention() string { return "<#" + s.String() + ">" }

// Mention generates the mention syntax for this role ID.
func (s RoleID) Mention() string { return "<@&" + s.String() + ">" }

// Mention generates the mention syntax for this user ID.
func (s UserID) Mention() string { return "<@" + s.String() + ">" }

// Snowflake is the format of Discord's ID type. It is a format that can be
// sorted chronologically.
type Snowflake uint64

// NullSnowflake gets encoded into a null. This is used for
// optional and nullable snowflake fields.
const NullSnowflake = ^Snowflake(0)

// NewSnowflake creates a new snowflake from the given time.
func NewSnowflake(t time.Time) Snowflake {
	return Snowflake((DurationSinceEpoch(t) / time.Millisecond) << 22)
}

// ParseSnowflake parses a snowflake.
func ParseSnowflake(sf string) (Snowflake, error) {
	if sf == "null" {
		return NullSnowflake, nil
	}

	u, err := strconv.ParseUint(sf, 10, 64)
	if err != nil {
		return 0, err
	}

	return Snowflake(u), nil
}

func (s *Snowflake) UnmarshalJSON(v []byte) error {
	p, err := ParseSnowflake(strings.Trim(string(v), `"`))
	if err != nil {
		return err
	}

	*s = p
	return nil
}

func (s Snowflake) MarshalJSON() ([]byte, error) {
	// This includes 0 and null, because MarshalJSON does not dictate when a
	// value gets omitted.
	if !s.IsValid() {
		return []byte("null"), nil
	} else {
		return []byte(`"` + strconv.FormatInt(int64(s), 10) + `"`), nil
	}
}

// String returns the ID, or nothing if the snowflake isn't valid.
func (s Snowflake) String() string {
	// Check if negative.
	if !s.IsValid() {
		return ""
	}
	return strconv.FormatUint(uint64(s), 10)
}

// IsValid returns whether or not the snowflake is valid.
func (s Snowflake) IsValid() bool {
	return !(int64(s) == 0 || s == NullSnowflake)
}

// IsNull returns whether or not the snowflake is null. This method is rarely
// ever useful; most people should use IsValid instead.
func (s Snowflake) IsNull() bool {
	return s == NullSnowflake
}

func (s Snowflake) Time() time.Time {
	unixnano := time.Duration(s>>22)*time.Millisecond + Epoch
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
