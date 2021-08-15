package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

var EndpointStageInstances = Endpoint + "stage-instances/"

// https://discord.com/developers/docs/resources/stage-instance#create-stage-instance-json-params
type CreateStageInstanceData struct {
	// ChannelID is the id of the Stage channel.
	ChannelID discord.ChannelID `json:"channel_id"`
	// Topic is the topic of the Stage instance (1-120 characters).
	Topic string `json:"topic"`
	// PrivacyLevel is the privacy level of the Stage instance.
	//
	// Defaults to discord.GuildOnlyStage.
	PrivacyLevel discord.PrivacyLevel `json:"privacy_level,omitempty"`

	AuditLogReason `json:"-"`
}

// CreateStageInstance creates a new Stage instance associated to a Stage
// channel.
//
// It requires the user to be a moderator of the Stage channel.
func (c *Client) CreateStageInstance(
	data CreateStageInstanceData) (*discord.StageInstance, error) {

	var s *discord.StageInstance
	return s, c.RequestJSON(
		&s, "POST",
		EndpointStageInstances,
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// https://discord.com/developers/docs/resources/stage-instance#update-stage-instance-json-params
type UpdateStageInstanceData struct {
	// Topic is the topic of the Stage instance (1-120 characters).
	Topic string `json:"topic,omitempty"`
	// PrivacyLevel is the privacy level of the Stage instance.
	PrivacyLevel discord.PrivacyLevel `json:"privacy_level,omitempty"`

	AuditLogReason `json:"-"`
}

// UpdateStageInstance updates fields of an existing Stage instance.
//
// It requires the user to be a moderator of the Stage channel.
func (c *Client) UpdateStageInstance(
	channelID discord.ChannelID, data UpdateStageInstanceData) error {

	return c.FastRequest(
		"PATCH",
		EndpointStageInstances+channelID.String(),
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

func (c *Client) DeleteStageInstance(channelID discord.ChannelID, reason AuditLogReason) error {
	return c.FastRequest(
		"DELETE", EndpointStageInstances+channelID.String(),
		httputil.WithHeaders(reason.Header()),
	)
}
