package discord

import "time"

type AutoModerationEventTypes uint32

const (
	AutoModerationMessageSend AutoModerationEventTypes = 1 + iota
	AutoModerationMemberUpdate
)

type AutoModerationTriggerTypes uint32

const (
	AutoModerationKeyword AutoModerationTriggerTypes = 1 + iota
	AutoModerationSpam
	AutoModerationKeywordPreset
	AutoModerationMentionSpam
	AutoModerationMemberProfile
)

type AutoModerationKeywordPresetTypes uint32

const (
	AutoModeratorProfanity = 1 + iota
	AutoModeratorSexualContent
	AutoModeratorSlurs
)

type AutoModerationRuleID Snowflake

// NullAutoModerationRuleID gets encoded into a null. This is used for optional and nullable snowflake fields.
const NullAutoModerationRuleID = AutoModerationRuleID(NullSnowflake)

func (s AutoModerationRuleID) MarshalJSON() ([]byte, error)  { return Snowflake(s).MarshalJSON() }
func (s *AutoModerationRuleID) UnmarshalJSON(v []byte) error { return (*Snowflake)(s).UnmarshalJSON(v) }

// String returns the ID, or nothing if the snowflake isn't valid.
func (s AutoModerationRuleID) String() string { return Snowflake(s).String() }

// IsValid returns whether or not the snowflake is valid.
func (s AutoModerationRuleID) IsValid() bool { return Snowflake(s).IsValid() }

// IsNull returns whether or not the snowflake is null. This method is rarely
// ever useful; most people should use IsValid instead.
func (s AutoModerationRuleID) IsNull() bool { return Snowflake(s).IsNull() }

func (s AutoModerationRuleID) Time() time.Time   { return Snowflake(s).Time() }
func (s AutoModerationRuleID) Worker() uint8     { return Snowflake(s).Worker() }
func (s AutoModerationRuleID) PID() uint8        { return Snowflake(s).PID() }
func (s AutoModerationRuleID) Increment() uint16 { return Snowflake(s).Increment() }

type AutoModerationTriggerMetadata struct {
	KeywordFilter                []string                           `json:"keyword_filter"`
	RegexPatterns                []string                           `json:"regex_patterns"`
	Presets                      []AutoModerationKeywordPresetTypes `json:"presets"`
	AllowList                    []string                           `json:"allow_list"`
	MentionTotalLimit            int                                `json:"mention_total_limit"`
	MentionRaidProtectionEnabled bool                               `json:"mention_raid_protection_enabled"`
}

type AutoModerationActionMetadata struct {
	ChannelID       ChannelID `json:"channel_id"`
	DurationSeconds int       `json:"duration_seconds"`
	CustomMessage   string    `json:"custom_message,omitempty"`
}

type AutoModerationActionTypes uint32

const (
	AutoModerationBlockMessage = 1 + iota
	AutoModerationSendAlertMessage
	AutoModerationTimeout
	AutoModerationBlockMemberInteraction
)

type AutoModerationAction struct {
	Type     AutoModerationActionTypes    `json:"type"`
	Metadata AutoModerationActionMetadata `json:"metadata,omitempty"`
}

type AutoModerationRule struct {
	ID              AutoModerationRuleID     `json:"id"`
	GuildID         GuildID                  `json:"guild_id"`
	Name            string                   `json:"name"`
	CreatorID       UserID                   `json:"creator_id"`
	EventType       AutoModerationEventTypes `json:"event_type"`
	TriggerType     AutoModerationTriggerTypes
	TriggerMetadata AutoModerationTriggerMetadata `json:"trigger_metadata"`
}
