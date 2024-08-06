package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

// Get a list of all rules currently configured for the guild. Returns a list of auto moderation rule objects for the given guild.
//
// This endpoint requires the MANAGE_GUILD permission.
func (c *Client) ListAutoModerationRules(guildID discord.GuildID) ([]discord.AutoModerationRule, error) {
	var rules []discord.AutoModerationRule
	return rules, c.RequestJSON(
		&rules, "GET",
		EndpointGuilds+guildID.String()+"/auto-moderation/rules",
	)
}

// Get a single rule. Returns an auto moderation rule object.
//
// This endpoint requires the MANAGE_GUILD permission.
func (c *Client) GetAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID) (discord.AutoModerationRule, error) {
	var rule discord.AutoModerationRule
	return rule, c.RequestJSON(
		&rule, "GET",
		EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String(),
	)
}

// Create a new rule. Returns an auto moderation rule on success. Fires an Auto Moderation Rule Create Gateway event.
//
// This endpoint requires the MANAGE_GUILD permission.
//
// This endpoint supports the X-Audit-Log-Reason header.
func (c *Client) CreateAutoModerationRule(guildID discord.GuildID, rule discord.AutoModerationRule) (*discord.AutoModerationRule, error) {
	var ret *discord.AutoModerationRule
	return ret, c.RequestJSON(&ret, "POST", EndpointGuilds+guildID.String()+"/auto-moderation/rules",
		httputil.WithJSONBody(rule),
	)
}

type ModifyAutoModerationRuleData struct {
	AuditLogReason
}

// Modify an existing rule. Returns an auto moderation rule on success. Fires an Auto Moderation Rule Update Gateway event.
//
// Requires MANAGE_GUILD permissions.
//
// All parameters for this endpoint are optional.
//
// This endpoint supports the X-Audit-Log-Reason header.
func (c *Client) ModifyAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID, rule discord.AutoModerationRule, data ModifyAutoModerationRuleData) (*discord.AutoModerationRule, error) {
	var ret *discord.AutoModerationRule
	return ret, c.RequestJSON(&ret, "PATCH", EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String(),
		httputil.WithJSONBody(rule),
		httputil.WithHeaders(data.Header()),
	)
}

type DeleteAutoModerationRuleData struct {
	AuditLogReason
}

// Delete a rule. Returns a 204 on success. Fires an Auto Moderation Rule Delete Gateway event.
//
// This endpoint requires the MANAGE_GUILD permission.
//
// This endpoint supports the X-Audit-Log-Reason header.
func (c *Client) DeleteAutoModerationRule(guildID discord.GuildID, ruleID discord.AutoModerationRuleID, data DeleteAutoModerationRuleData) error {
	return c.FastRequest("DELETE", EndpointGuilds+guildID.String()+"/auto-moderation/rules/"+ruleID.String(), httputil.WithHeaders(data.Header()))
}
