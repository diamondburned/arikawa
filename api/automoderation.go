package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

func (c *Client) ListAutoModerationRules(guildID discord.GuildID) ([]discord.AutoModerationRule, error) {
	var rules []discord.AutoModerationRule
	return rules, c.RequestJSON(
		&rules, "GET",
		EndpointGuilds+guildID.String()+"/auto-moderation/rules",
	)
}

func (c *Client) GetAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID) (discord.AutoModerationRule, error) {
	var rule discord.AutoModerationRule
	return rule, c.RequestJSON(
		&rule, "GET",
		EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String(),
	)
}

type CreateAutomoderationRulePayload struct {
	Name            string                                `json:"name"`
	EventType       discord.AutoModerationEventTypes      `json:"event_type"`
	TriggerType     discord.AutoModerationTriggerTypes    `json:"trigger_type"`
	TriggerMetadata discord.AutoModerationTriggerMetadata `json:"trigger_metadata,omitempty"`
	Actions         []discord.AutoModerationAction        `json:"actions"`
	Enabled         bool                                  `json:"enabled,omitempty"`
	ExemptRoles     []discord.RoleID                      `json:"exempt_roles,omitempty"`
	ExemptChannels  []discord.ChannelID                   `json:"exempt_channels,omitempty"`
}

func (c *Client) CreateAutoModerationRule(guildID discord.GuildID, rule CreateAutomoderationRulePayload) (*discord.AutoModerationRule, error) {
	var ret *discord.AutoModerationRule
	return ret, c.RequestJSON(&ret, "POST", EndpointGuilds+guildID.String()+"/auto-moderation/rules",
		httputil.WithJSONBody(rule),
	)
}

func (c *Client) ModifyAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID, rule CreateAutomoderationRulePayload) (*discord.AutoModerationRule, error) {
	var ret *discord.AutoModerationRule
	return ret, c.RequestJSON(&ret, "PATCH", EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String(),
		httputil.WithJSONBody(rule),
	)
}

func (c *Client) DeleteAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID) error {
	return c.FastRequest("DELETE", EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String())
}
