package discord

// A StageInstance holds information about a live stage instance.
//
// https://discord.com/developers/docs/resources/stage-instance#stage-instance-object
type StageInstance struct {
	// ID is the id of this Stage instance.
	ID StageID `json:"id"`
	// GuildID is the guild id of the associated Stage channel.
	GuildID GuildID `json:"guild_id"`
	// ChannelID is the id of the associated Stage channel.
	ChannelID ChannelID `json:"channel_id"`
	// Topic is the topic of the Stage instance (1-120 characters).
	Topic string `json:"topic"`
	// PrivacyLevel is the privacy level of the Stage instance.
	PrivacyLevel PrivacyLevel `json:"privacy_level"`
	// NotDiscoverable defines whether or not Stage discovery is disabled.
	NotDiscoverable bool `json:"discoverable_disabled"`
}

type PrivacyLevel int

// https://discord.com/developers/docs/resources/stage-instance#stage-instance-object-privacy-level
const (
	// PublicStage is used if a StageInstance instance is visible publicly, such as on
	// StageInstance discovery.
	PublicStage PrivacyLevel = iota + 1
	// GuildOnlyStage is used if a StageInstance instance is visible to only guild
	// members.
	GuildOnlyStage
)
