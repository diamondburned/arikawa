package discord

// EventStatus describes the different statuses GuildScheduledEvent can be.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-status
type EventStatus int

const (
	ScheduledEvent EventStatus = iota + 1
	ActiveEvent
	CompletedEvent
	CancelledEvent
)

// EntityType describes the different types GuildScheduledEvent can be.
type EntityType int

const (
	StageInstanceEntity EntityType = iota + 1
	VoiceEntity
	ExternalEntity
)

// ScheduledEventPrivacy describes the privacy levels of GuildScheduledEvent.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-privacy-level
type ScheduledEventPrivacyLevel int

const (
	// GuildOnly requires the scheduled event to be only accessible to guild members.
	GuildOnly ScheduledEventPrivacyLevel = iota + 2
)

// GuildScheduledEvent describes the scheduled event structure.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-structure
type GuildScheduledEvent struct {
	// ID is the id of the scheduled event.
	ID EventID `json:"id"`
	// GuildID is the guild id of where the scheduled event belongs to.
	GuildID GuildID `json:"guild_id"`
	// ChannelID is the channel id in which the scheduled event will be
	// hosted at, this may be NullChannelID if the EntityType is set
	// to ExternalEntity.
	ChannelID ChannelID `json:"channel_id"`
	// CreatorID is the user id of who created the scheduled event.
	CreatorID UserID `json:"creator_id"`
	// Name is the name of the scheduled event.
	Name string `json:"name"`
	// Description is the description of the scheduled event.
	Description string `json:"description"`
	// StartTime is when the scheduled event will start at.
	StartTime Timestamp `json:"scheduled_start_time"`
	// EndTime is when the scheduled event will end at, if it does.
	EndTime Timestamp `json:"scheduled_end_time"`
	// PrivacyLevel is the privacy level of the scheduled event.
	PrivacyLevel ScheduledEventPrivacyLevel `json:"privacy_level"`
	// Status is the status of the scheduled event.
	Status EventStatus `json:"status"`
	// EntityType describes the type of scheduled event.
	EntityType EntityType `json:"entity_type"`
	// EntityID is the id of an entity associated with a scheduled event.
	EntityID EntityID `json:"entity_id"`
	// EntityMetadata is additional metadata for the scheduled event.
	EntityMetadata *EntityMetadata `json:"entity_metadata"`
	// Creator is the the user responsible for creating the scheduled event. This field
	// will only be present if CreatorID is
	Creator *User `json:"creator"`
	// UserCount is the number of users subscribed to the scheduled event.
	UserCount int `json:"user_count"`
	// Image is the cover image hash of the scheduled event.
	Image Hash `json:"image,omitempty"`
}

// EntityMetadata is the entity metadata of GuildScheduledEvent.
type EntityMetadata struct {
	// Location describes where the event takes place at. This is not
	// optional when GuildScheduled#EntityType is set as ExternalEntity.
	Location string `json:"location,omitempty"`
}
