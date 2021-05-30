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

type Snowflake uint64

// NullSnowflake gets encoded into a null. This is used for
// optional and nullable snowflake fields.
const NullSnowflake = ^Snowflake(0)

func NewSnowflake(t time.Time) Snowflake {
	return Snowflake((DurationSinceEpoch(t) / time.Millisecond) << 22)
}

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

// IsNull returns whether or not the snowflake is null.
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

type AppID Snowflake

const NullAppID = AppID(NullSnowflake)

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

const NullAttachmentID = AttachmentID(NullSnowflake)

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

const NullAuditLogEntryID = AuditLogEntryID(NullSnowflake)

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

const NullChannelID = ChannelID(NullSnowflake)

func (s ChannelID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *ChannelID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s ChannelID) String() string                { return Snowflake(s).String() }
func (s ChannelID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s ChannelID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s ChannelID) Time() time.Time               { return Snowflake(s).Time() }
func (s ChannelID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s ChannelID) PID() uint8                    { return Snowflake(s).PID() }
func (s ChannelID) Increment() uint16             { return Snowflake(s).Increment() }
func (s ChannelID) Mention() string               { return "<#" + s.String() + ">" }

type CommandID Snowflake

const NullCommandID = CommandID(NullSnowflake)

func (s CommandID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *CommandID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s CommandID) String() string                { return Snowflake(s).String() }
func (s CommandID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s CommandID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s CommandID) Time() time.Time               { return Snowflake(s).Time() }
func (s CommandID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s CommandID) PID() uint8                    { return Snowflake(s).PID() }
func (s CommandID) Increment() uint16             { return Snowflake(s).Increment() }

type EmojiID Snowflake

const NullEmojiID = EmojiID(NullSnowflake)

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

const NullIntegrationID = IntegrationID(NullSnowflake)

func (s IntegrationID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *IntegrationID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s IntegrationID) String() string                { return Snowflake(s).String() }
func (s IntegrationID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s IntegrationID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s IntegrationID) Time() time.Time               { return Snowflake(s).Time() }
func (s IntegrationID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s IntegrationID) PID() uint8                    { return Snowflake(s).PID() }
func (s IntegrationID) Increment() uint16             { return Snowflake(s).Increment() }

type InteractionID Snowflake

const NullInteractionID = InteractionID(NullSnowflake)

func (s InteractionID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *InteractionID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s InteractionID) String() string                { return Snowflake(s).String() }
func (s InteractionID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s InteractionID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s InteractionID) Time() time.Time               { return Snowflake(s).Time() }
func (s InteractionID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s InteractionID) PID() uint8                    { return Snowflake(s).PID() }
func (s InteractionID) Increment() uint16             { return Snowflake(s).Increment() }

type GuildID Snowflake

const NullGuildID = GuildID(NullSnowflake)

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

const NullMessageID = MessageID(NullSnowflake)

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

const NullRoleID = RoleID(NullSnowflake)

func (s RoleID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *RoleID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s RoleID) String() string                { return Snowflake(s).String() }
func (s RoleID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s RoleID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s RoleID) Time() time.Time               { return Snowflake(s).Time() }
func (s RoleID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s RoleID) PID() uint8                    { return Snowflake(s).PID() }
func (s RoleID) Increment() uint16             { return Snowflake(s).Increment() }
func (s RoleID) Mention() string               { return "<@&" + s.String() + ">" }

type StageID Snowflake

const NullStageID = StageID(NullSnowflake)

func (s StageID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *StageID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s StageID) String() string                { return Snowflake(s).String() }
func (s StageID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s StageID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s StageID) Time() time.Time               { return Snowflake(s).Time() }
func (s StageID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s StageID) PID() uint8                    { return Snowflake(s).PID() }
func (s StageID) Increment() uint16             { return Snowflake(s).Increment() }

type StickerID Snowflake

const NullStickerID = StickerID(NullSnowflake)

func (s StickerID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *StickerID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s StickerID) String() string                { return Snowflake(s).String() }
func (s StickerID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s StickerID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s StickerID) Time() time.Time               { return Snowflake(s).Time() }
func (s StickerID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s StickerID) PID() uint8                    { return Snowflake(s).PID() }
func (s StickerID) Increment() uint16             { return Snowflake(s).Increment() }

type StickerPackID Snowflake

const NullStickerPackID = StickerPackID(NullSnowflake)

func (s StickerPackID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *StickerPackID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s StickerPackID) String() string                { return Snowflake(s).String() }
func (s StickerPackID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s StickerPackID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s StickerPackID) Time() time.Time               { return Snowflake(s).Time() }
func (s StickerPackID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s StickerPackID) PID() uint8                    { return Snowflake(s).PID() }
func (s StickerPackID) Increment() uint16             { return Snowflake(s).Increment() }

type UserID Snowflake

const NullUserID = UserID(NullSnowflake)

func (s UserID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *UserID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s UserID) String() string                { return Snowflake(s).String() }
func (s UserID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s UserID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s UserID) Time() time.Time               { return Snowflake(s).Time() }
func (s UserID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s UserID) PID() uint8                    { return Snowflake(s).PID() }
func (s UserID) Increment() uint16             { return Snowflake(s).Increment() }
func (s UserID) Mention() string               { return "<@" + s.String() + ">" }

type WebhookID Snowflake

const NullWebhookID = WebhookID(NullSnowflake)

func (s WebhookID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *WebhookID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }
func (s WebhookID) String() string                { return Snowflake(s).String() }
func (s WebhookID) IsValid() bool                 { return Snowflake(s).IsValid() }
func (s WebhookID) IsNull() bool                  { return Snowflake(s).IsNull() }
func (s WebhookID) Time() time.Time               { return Snowflake(s).Time() }
func (s WebhookID) Worker() uint8                 { return Snowflake(s).Worker() }
func (s WebhookID) PID() uint8                    { return Snowflake(s).PID() }
func (s WebhookID) Increment() uint16             { return Snowflake(s).Increment() }
