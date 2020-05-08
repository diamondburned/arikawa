package discord

import "github.com/diamondburned/arikawa/utils/json"

type AuditLog struct {
	// List of webhooks found in the audit log
	Webhooks []Webhook `json:"webhooks"`
	// List of users found in the audit log
	Users []User `json:"users"`
	// List of audit log entries
	Entries []AuditLogEntries `json:"audit_log_entries"`
	// List of partial integration objects, only ID, Name, Type, and Account
	Integrations []Integration `json:"integrations"`
}

type AuditLogEntries struct {
	ID       Snowflake `json:"id"`
	UserID   Snowflake `json:"user_id"`
	TargetID string    `json:"target_id,omitempty"`

	ActionType AuditLogEvent `json:"action_type"`

	Changes []AuditLogChange `json:"changes,omitempty"`
	Options AuditEntryInfo   `json:"options,omitempty"`
	Reason  string           `json:"reason,omitempty"`
}

type AuditLogEvent uint8

const (
	GuildUpdate            AuditLogEvent = 1
	ChannelCreate          AuditLogEvent = 10
	ChannelUpdate          AuditLogEvent = 11
	ChannelDelete          AuditLogEvent = 12
	ChannelOverwriteCreate AuditLogEvent = 13
	ChannelOverwriteUpdate AuditLogEvent = 14
	ChannelOverwriteDelete AuditLogEvent = 15
	MemberKick             AuditLogEvent = 20
	MemberPrune            AuditLogEvent = 21
	MemberBanAdd           AuditLogEvent = 22
	MemberBanRemove        AuditLogEvent = 23
	MemberUpdate           AuditLogEvent = 24
	MemberRoleUpdate       AuditLogEvent = 25
	MemberMove             AuditLogEvent = 26
	MemberDisconnect       AuditLogEvent = 27
	BotAdd                 AuditLogEvent = 28
	RoleCreate             AuditLogEvent = 30
	RoleUpdate             AuditLogEvent = 31
	RoleDelete             AuditLogEvent = 32
	InviteCreate           AuditLogEvent = 40
	InviteUpdate           AuditLogEvent = 41
	InviteDelete           AuditLogEvent = 42
	WebhookCreate          AuditLogEvent = 50
	WebhookUpdate          AuditLogEvent = 51
	WebhookDelete          AuditLogEvent = 52
	EmojiCreate            AuditLogEvent = 60
	EmojiUpdate            AuditLogEvent = 61
	EmojiDelete            AuditLogEvent = 62
	MessageDelete          AuditLogEvent = 72
	MessageBulkDelete      AuditLogEvent = 73
	MessagePin             AuditLogEvent = 74
	MessageUnpin           AuditLogEvent = 75
	IntegrationCreate      AuditLogEvent = 80
	IntegrationUpdate      AuditLogEvent = 81
	IntegrationDelete      AuditLogEvent = 82
)

type AuditEntryInfo struct {
	// MEMBER_PRUNE
	DeleteMemberDays string `json:"delete_member_days,omitempty"`
	// MEMBER_PRUNE
	MembersRemoved string `json:"members_removed,omitempty"`
	// MEMBER_MOVE & MESSAGE_PIN & MESSAGE_UNPIN & MESSAGE_DELETE
	ChannelID Snowflake `json:"channel_id,omitempty"`
	// MESSAGE_PIN & MESSAGE_UNPIN
	MessageID Snowflake `json:"message_id,omitempty"`
	// MESSAGE_DELETE & MESSAGE_BULK_DELETE & MEMBER_DISCONNECT & MEMBER_MOVE
	Count string `json:"count,omitempty"`
	// CHANNEL_OVERWRITE_CREATE & CHANNEL_OVERWRITE_UPDATE & CHANNEL_OVERWRITE_DELETE
	ID Snowflake `json:"id,omitempty"`
	// CHANNEL_OVERWRITE_CREATE & CHANNEL_OVERWRITE_UPDATE & CHANNEL_OVERWRITE_DELETE
	Type ChannelOverwritten `json:"type,omitempty"`
	// CHANNEL_OVERWRITE_CREATE & CHANNEL_OVERWRITE_UPDATE & CHANNEL_OVERWRITE_DELETE
	RoleName string `json:"role_name,omitempty"`
}

// ChannelOverwritten is the type of overwritten entity in
// (AuditEntryInfo).Type.
type ChannelOverwritten string

const (
	MemberChannelOverwritten ChannelOverwritten = "member"
	RoleChannelOverwritten   ChannelOverwritten = "role"
)

type AuditLogChange struct {
	Key      string             `json:"key"`
	NewValue *AuditLogChangeKey `json:"new_value,omitempty"`
	OldValue *AuditLogChangeKey `json:"old_value,omitempty"`
}

type AuditLogChangeKey struct {
	// The ID of the changed entity - sometimes used in conjunction with other
	// keys
	ID Snowflake `json:"snowflake"`
	// Type of entity created, either a ChannelType (int) or string.
	Type json.AlwaysString `json:"type"`

	*AuditLogChangeGuild
	*AuditLogChangeChannel
	*AuditLogChangeRole
	*AuditLogChangeInvite
	*AuditLogChangeUser
	*AuditLogChangeIntegration
}

// AuditLogChangeGuild is the audit log key for Guild.
type AuditLogChangeGuild struct {
	// Name changed
	Name string `json:"name,omitempty"`
	// Icon changed
	IconHash string `json:"icon_hash,omitempty"`
	// Invite splash page artwork changed
	SplashHash string `json:"splash_hash,omitempty"`
	// Owner changed
	OwnerID Snowflake `json:"owner_id,omitempty"`
	// Region changed
	Region string `json:"region,omitempty"`
	// AFK channel changed
	AfkChannelID Snowflake `json:"afk_channel_id,omitempty"`
	// AFK timeout duration changed
	AFKTimeout Seconds `json:"afk_timeout,omitempty"`
	// Two-factor auth requirement changed
	MFA MFALevel `json:"mfa_level,omitempty"`
	// Required verification level changed
	Verification Verification `json:"verification_level,omitempty"`
	// Change in whose messages are scanned and deleted for explicit content in
	// the server
	ExplicitFilter ExplicitFilter `json:"explicit_content_filter,omitempty"`
	// Default message notification level changed
	Notification Notification `json:"default_message_notifications,omitempty"`
	// Guild invite vanity url changed
	VanityURLCode string `json:"vanity_url_code,omitempty"`
	// New role added, only ID and Name are available
	RoleAdd []Role `json:"$add,omitempty"`
	// Role removed, partial similar to RoleAdd
	RoleRemove []Role `json:"$remove,omitempty"`
	// Change in number of days after which inactive and role-unassigned members
	// are kicked
	PruneDeleteDays int `json:"prune_delete_days,omitempty"`
	// Server widget enabled/disable
	WidgetEnabled bool `json:"widget_enabled,omitempty"`
	// Channel id of the server widget changed
	WidgetChannelID Snowflake `json:"widget_channel_id,omitempty"`
	// ID of the system channel changed
	SystemChannelID Snowflake `json:"system_channel_id,omitempty"`
}

// AuditLogChangeChannel is the audit log key for Channel.
type AuditLogChangeChannel struct {
	// Text channel topic changed
	Topic string `json:"topic,omitempty"`
	// Voice channel bitrate changed
	Bitrate uint `json:"bitrate,omitempty"`
	// Permissions on a channel changed
	Permissions []Overwrite `json:"permission_overwrites,omitempty"`
	// Channel NSFW restriction changed
	NSFW bool `json:"nsfw,omitempty"`
	// Application ID of the added or removed webhook or bot
	ApplicationID Snowflake `json:"application_id,omitempty"`
	// Amount of seconds a user has to wait before sending another message
	// changed
	UserRateLimit Seconds `json:"rate_limit_per_user,omitempty"`
}

// AuditLogChangeRole is the audit log key for Role.
type AuditLogChangeRole struct {
	// Permissions for a role changed
	Permissions Permissions `json:"permissions,omitempty"`
	// Role color changed
	Color Color `json:"color,omitempty"`
	// Role is now displayed/no longer displayed separate from online users
	Hoist bool `json:"hoist,omitempty"`
	// Role is now mentionable/unmentionable
	Mentionable bool `json:"mentionable,omitempty"`
	// A permission on a text or voice channel was allowed for a role
	Allow Permissions `json:"allow,omitempty"`
	// A permission on a text or voice channel was denied for a role
	Deny Permissions `json:"deny,omitempty"`
}

// AuditLogChangeInvite is the audit log key for InviteMetadata.
type AuditLogChangeInvite struct {
	// Invite code changed
	Code string `json:"code,omitempty"`
	// Channel for invite code changed
	ChannelID Snowflake `json:"channel_id,omitempty"`
	// Person who created invite code changed
	InviterID Snowflake `json:"inviter_id,omitempty"`
	// Change to max number of times invite code can be used
	MaxUses int `json:"max_uses,omitempty"`
	// Number of times invite code used changed
	Uses int `json:"uses,omitempty"`
	// How long invite code lasts changed
	MaxAge Seconds `json:"max_age,omitempty"`
	// Invite code is temporary/never expires
	Temporary bool `json:"temporary,omitempty"`
}

// AuditLogChangeUser is the audit log key for User.
type AuditLogChangeUser struct {
	// User server deafened/undeafened
	Deaf bool `json:"deaf,omitempty"`
	// User server muted/unmuted
	Mute bool `json:"mute,omitempty"`
	// User nickname changed
	Nick string `json:"nick,omitempty"`
	// User avatar changed
	Avatar Hash `json:"avatar_hash,omitempty"`
}

// AuditLogChangeIntegration is the audit log key for Integration.
type AuditLogChangeIntegration struct {
	// Integration emoticons enabled/disabled
	EnableEmoticons bool `json:"enable_emoticons,omitempty"`
	// Integration expiring subscriber behavior changed
	ExpireBehavior int `json:"expire_behavior,omitempty"`
	// Integration expire grace period changed
	ExpireGracePeriod int `json:"expire_grace_period,omitempty"`
}
