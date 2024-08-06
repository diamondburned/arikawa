package discord

type AutoModerationEventType uint32

const (
	AutoModerationMessageSend AutoModerationEventType = 1 + iota
	AutoModerationMemberUpdate
)

type AutoModerationTriggerType uint32

const (
	AutoModerationKeyword AutoModerationTriggerType = 1 + iota
	AutoModerationSpam
	AutoModerationKeywordPreset
	AutoModerationMentionSpam
	AutoModerationMemberProfile
)

type AutoModerationKeywordPresetType uint32

const (
	AutoModeratorProfanity AutoModerationKeywordPresetType = 1 + iota
	AutoModeratorSexualContent
	AutoModeratorSlurs
)

type AutoModerationTriggerMetadata struct {
	// substrings which will be searched for in content (Maximum of 1000)
	KeywordFilter []string `json:"keyword_filter"`
	// regular expression patterns which will be matched against content (Maximum of 10)
	RegexPatterns []string `json:"regex_patterns"`
	// the internally pre-defined wordsets which will be searched for in content
	Presets []AutoModerationKeywordPresetType `json:"presets"`
	// substrings which should not trigger the rule (Maximum of 100 or 1000)
	AllowList []string `json:"allow_list"`
	// total number of unique role and user mentions allowed per message (Maximum of 50)
	MentionTotalLimit int `json:"mention_total_limit"`
	// whether to automatically detect mention raids
	MentionRaidProtectionEnabled bool `json:"mention_raid_protection_enabled"`
}

type AutoModerationActionMetadata struct {
	// channel to which user content should be logged
	ChannelID ChannelID `json:"channel_id"`
	// timeout duration in seconds
	// maximum of 2419200 seconds (4 weeks)
	DurationSeconds int `json:"duration_seconds"`
	// additional explanation that will be shown to members whenever their message is blocked
	// maximum of 150 characters
	CustomMessage string `json:"custom_message,omitempty"`
}

type AutoModerationActionType uint32

const (
	AutoModerationBlockMessage AutoModerationActionType = 1 + iota
	AutoModerationSendAlertMessage
	AutoModerationTimeout
	AutoModerationBlockMemberInteraction
)

type AutoModerationAction struct {
	// the type of action
	Type AutoModerationActionType `json:"type"`
	// additional metadata needed during execution for this specific action type
	Metadata AutoModerationActionMetadata `json:"metadata,omitempty"`
}

type AutoModerationRule struct {
	// the id of this rule
	ID AutoModerationRuleID `json:"id"`
	// the id of the guild which this rule belongs to
	GuildID GuildID `json:"guild_id"`
	// the rule name
	Name string `json:"name"`
	// the user which first created this rule
	CreatorID UserID `json:"creator_id,omitempty"`
	// the rule event type
	EventType AutoModerationEventType `json:"event_type"`
	// the rule trigger type
	TriggerType AutoModerationTriggerType
	// the rule trigger metadata
	TriggerMetadata AutoModerationTriggerMetadata `json:"trigger_metadata,omitempty"`
	// the actions which will execute when the rule is triggered
	Actions []AutoModerationAction `json:"actions"`
	// whether the rule is enabled
	Enabled bool `json:"enabled,omitempty"`
	// the role ids that should not be affected by the rule (Maximum of 20)
	ExemptRoles []RoleID `json:"exempt_roles,omitempty"`
	// the channel ids that should not be affected by the rule (Maximum of 50)
	ExemptChannels []ChannelID `json:"exempt_channels,omitempty"`
}
