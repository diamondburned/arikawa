package discord

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
