package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// CreateScheduledEventData is the structure for creating a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#create-guild-scheduled-event-json-params
type CreateScheduledEventData struct {
	// ChannelID is the channel id of the scheduled event.
	ChannelID discord.ChannelID `json:"channel_id"`
	// EntityMetadata is the entity metadata of the scheduled event.
	EntityMetadata *discord.EntityMetadata `json:"entity_metadata"`
	// Name is the name of the scheduled event.
	Name string `json:"name"`
	// PrivacyLevel is the privacy level of the scheduled event.
	PrivacyLevel discord.ScheduledEventPrivacyLevel `json:"privacy_level"`
	// StartTime is when the scheduled event begins.
	StartTime discord.Timestamp `json:"scheduled_start_time"`
	// EndTime is when the scheduled event ends, if it does.
	EndTime *discord.Timestamp `json:"scheduled_end_time,omitempty"`
	// Description is the description of the schduled event.
	Description string `json:"description"`
	// EntityType is the entity type of the scheduled event.
	EntityType discord.EntityType `json:"entity_type"`
	// Image is the cover image of the scheduled event.
	Image Image `json:"image"`
}

// EditScheduledEventData is the structure for modifying a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#modify-guild-scheduled-event-json-params
type EditScheduledEventData struct {
	// ChannelID is the new channel id of the scheduled event.
	ChannelID discord.ChannelID `json:"channel_id,omitempty"`
	// EntityMetadata is the new entity metadata of the scheduled event.
	EntityMetadata *discord.EntityMetadata `json:"entity_metadata,omitempty"`
	// Name is the new name of the scheduled event.
	Name option.NullableString `json:"name,omitempty"`
	// PrivacyLevel is the new privacy level of the scheduled event.
	PrivacyLevel discord.ScheduledEventPrivacyLevel `json:"privacy_level,omitempty"`
	// StartTime is the new starting time for when the scheduled event begins.
	StartTime *discord.Timestamp `json:"scheduled_start_time,omitempty"`
	// EndTime is the new time of which the scheduled event ends
	EndTime *discord.Timestamp `json:"scheduled_end_time,omitempty"`
	// Description is the new description of the scheduled event.
	Description option.NullableString `json:"description,omitempty"`
	// EntityType is the new entity type of the scheduled event.
	EntityType discord.EntityType `json:"entity_type,omitempty"`
	// Status is the new event status of the scheduled event.
	Status discord.EventStatus `json:"status,omitempty"`
	// Image is the new image of the scheduled event.
	Image *Image `json:"image,omitempty"`
}

// GuildScheduledEventUser represents a user interested in a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-user-object
type GuildScheduledEventUser struct {
	// EventID is the id of the scheduled event.
	EventID discord.EventID `json:"guild_scheduled_event_id"`
	// User is the user object of the user.
	User discord.User `json:"user"`
	// Member is the member object of the user.
	Member *discord.Member `json:"member"`
}

// ListScheduledEventUsers returns a list of users currently in a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#get-guild-scheduled-event-users
func (c *Client) ListScheduledEventUsers(
	guildID discord.GuildID, eventID discord.EventID, limit option.NullableInt,
	withMember bool, before, after discord.UserID) ([]*GuildScheduledEventUser, error) {
	var eventUsers []*GuildScheduledEventUser
	var params struct {
		Limit      option.NullableInt `schema:"limit,omitempty"`
		WithMember bool               `schema:"with_member,omitempty"`
		Before     discord.UserID     `schema:"before,omitempty"`
		After      discord.UserID     `schema:"after,omitempty"`
	}
	params.Limit = limit
	params.WithMember = withMember
	params.Before = before
	params.After = after

	return eventUsers, c.RequestJSON(
		&eventUsers, "GET", EndpointGuilds+guildID.String()+"/scheduled-events/"+eventID.String()+"/users",
		httputil.WithSchema(c, params),
	)

}

// ListScheduledEvents lists the scheduled events in a guild.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#get-guild-scheduled-event-users
func (c *Client) ListScheduledEvents(guildID discord.GuildID, withUserCount bool) ([]*discord.GuildScheduledEvent, error) {
	var scheduledEvents []*discord.GuildScheduledEvent
	var params struct {
		WithUserCount bool `schema:"with_user_count"`
	}
	params.WithUserCount = withUserCount
	return scheduledEvents, c.RequestJSON(
		&scheduledEvents, "GET", EndpointGuilds+guildID.String()+"/scheduled-events",
		httputil.WithSchema(c, params),
	)
}

// CreateScheduledEvent creates a new scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#create-guild-scheduled-event
func (c *Client) CreateScheduledEvent(guildID discord.GuildID, reason AuditLogReason,
	data CreateScheduledEventData) (*discord.GuildScheduledEvent, error) {
	var scheduledEvent *discord.GuildScheduledEvent
	return scheduledEvent, c.RequestJSON(
		&scheduledEvent, "POST",
		EndpointGuilds+guildID.String()+"/scheduled-events",
		httputil.WithJSONBody(data),
		httputil.WithHeaders(reason.Header()),
	)
}

// EditScheduledEvent modifies the attributes of a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#modify-guild-scheduled-event
func (c *Client) EditScheduledEvent(guildID discord.GuildID, eventID discord.EventID, reason AuditLogReason,
	data EditScheduledEventData) (*discord.GuildScheduledEvent, error) {
	var modifiedEvent *discord.GuildScheduledEvent
	return modifiedEvent, c.RequestJSON(
		&modifiedEvent,
		"PATCH", EndpointGuilds+guildID.String()+"/scheduled-events/"+eventID.String(),
		httputil.WithHeaders(reason.Header()),
		httputil.WithJSONBody(data),
	)
}

// DeleteScheduledEvent deletes a scheduled event.
//
// https://discord.com/developers/docs/resources/guild-scheduled-event#delete-guild-scheduled-event
func (c *Client) DeleteScheduledEvent(guildID discord.GuildID, eventID discord.EventID) error {
	return c.FastRequest(
		"DELETE", EndpointGuilds+guildID.String()+"/scheduled-events/"+eventID.String(),
	)
}
