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
	if sf == "null" {
		return NullSnowflake, nil
	}

	i, err := strconv.ParseInt(sf, 10, 64)
	if err != nil {
		return 0, err
	}

	return Snowflake(i), nil
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
	return int64(s) > 0
}

// IsNull returns whether or not the snowflake is null.
func (s Snowflake) IsNull() bool {
	return s == NullSnowflake
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

type AppID Snowflake

func (s AppID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *AppID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s AppID) String() string                { return Snowflake(s).String() }
func (s AppID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s AppID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s AppID) Time() time.Time               { return Snowflake(s).Time() }
func (s AppID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s AppID) PID() uint8                    { return Snowflake(s).PID() }
func (s AppID) Increment() uint16             { return Snowflake(s).Increment() }

type AttachmentID Snowflake

func (s AttachmentID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *AttachmentID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s AttachmentID) String() string                { return Snowflake(s).String() }
func (s AttachmentID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s AttachmentID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s AttachmentID) Time() time.Time               { return Snowflake(s).Time() }
func (s AttachmentID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s AttachmentID) PID() uint8                    { return Snowflake(s).PID() }
func (s AttachmentID) Increment() uint16             { return Snowflake(s).Increment() }

type AuditLogEntryID Snowflake

func (s AuditLogEntryID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *AuditLogEntryID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s AuditLogEntryID) String() string                { return Snowflake(s).String() }
func (s AuditLogEntryID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s AuditLogEntryID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s AuditLogEntryID) Time() time.Time               { return Snowflake(s).Time() }
func (s AuditLogEntryID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s AuditLogEntryID) PID() uint8                    { return Snowflake(s).PID() }
func (s AuditLogEntryID) Increment() uint16             { return Snowflake(s).Increment() }

type ChannelID Snowflake

func (s ChannelID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *ChannelID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s ChannelID) String() string                { return Snowflake(s).String() }
func (s ChannelID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s ChannelID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s ChannelID) Time() time.Time               { return Snowflake(s).Time() }
func (s ChannelID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s ChannelID) PID() uint8                    { return Snowflake(s).PID() }
func (s ChannelID) Increment() uint16             { return Snowflake(s).Increment() }

type EmojiID Snowflake

func (s EmojiID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *EmojiID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s EmojiID) String() string                { return Snowflake(s).String() }
func (s EmojiID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s EmojiID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s EmojiID) Time() time.Time               { return Snowflake(s).Time() }
func (s EmojiID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s EmojiID) PID() uint8                    { return Snowflake(s).PID() }
func (s EmojiID) Increment() uint16             { return Snowflake(s).Increment() }

type IntegrationID Snowflake

func (s IntegrationID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *IntegrationID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s IntegrationID) String() string                { return Snowflake(s).String() }
func (s IntegrationID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s IntegrationID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s IntegrationID) Time() time.Time               { return Snowflake(s).Time() }
func (s IntegrationID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s IntegrationID) PID() uint8                    { return Snowflake(s).PID() }
func (s IntegrationID) Increment() uint16             { return Snowflake(s).Increment() }

type GuildID Snowflake

func (s GuildID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *GuildID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s GuildID) String() string                { return Snowflake(s).String() }
func (s GuildID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s GuildID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s GuildID) Time() time.Time               { return Snowflake(s).Time() }
func (s GuildID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s GuildID) PID() uint8                    { return Snowflake(s).PID() }
func (s GuildID) Increment() uint16             { return Snowflake(s).Increment() }

type MessageID Snowflake

func (s MessageID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *MessageID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s MessageID) String() string                { return Snowflake(s).String() }
func (s MessageID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s MessageID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s MessageID) Time() time.Time               { return Snowflake(s).Time() }
func (s MessageID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s MessageID) PID() uint8                    { return Snowflake(s).PID() }
func (s MessageID) Increment() uint16             { return Snowflake(s).Increment() }

type RoleID Snowflake

func (s RoleID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *RoleID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s RoleID) String() string                { return Snowflake(s).String() }
func (s RoleID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s RoleID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s RoleID) Time() time.Time               { return Snowflake(s).Time() }
func (s RoleID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s RoleID) PID() uint8                    { return Snowflake(s).PID() }
func (s RoleID) Increment() uint16             { return Snowflake(s).Increment() }

type UserID Snowflake

func (s UserID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *UserID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s UserID) String() string                { return Snowflake(s).String() }
func (s UserID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s UserID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s UserID) Time() time.Time               { return Snowflake(s).Time() }
func (s UserID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s UserID) PID() uint8                    { return Snowflake(s).PID() }
func (s UserID) Increment() uint16             { return Snowflake(s).Increment() }

type WebhookID Snowflake

func (s WebhookID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *WebhookID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s WebhookID) String() string                { return Snowflake(s).String() }
func (s WebhookID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s WebhookID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s WebhookID) Time() time.Time               { return Snowflake(s).Time() }
func (s WebhookID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s WebhookID) PID() uint8                    { return Snowflake(s).PID() }
func (s WebhookID) Increment() uint16             { return Snowflake(s).Increment() }
