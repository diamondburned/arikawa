package discord

import (
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/pkg/errors"
)

type AuditLog struct {
	// List of webhooks found in the audit log
	Webhooks []Webhook `json:"webhooks"`
	// List of users found in the audit log
	Users []User `json:"users"`
	// List of audit log entries
	Entries []AuditLogEntry `json:"audit_log_entries"`
	// List of partial integration objects, only ID, Name, Type, and Account
	Integrations []Integration `json:"integrations"`
}

// AuditLogEntry is a single entry in the audit log.
type AuditLogEntry struct {
	ID       Snowflake `json:"id"`
	UserID   Snowflake `json:"user_id"`
	TargetID string    `json:"target_id,omitempty"`

	ActionType AuditLogEvent `json:"action_type"`

	Changes []AuditLogChange `json:"changes,omitempty"`
	Options AuditEntryInfo   `json:"options,omitempty"`
	Reason  string           `json:"reason,omitempty"`
}

// AuditLogEvent is the type of audit log action that occured.
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

// AuditLogChange is a single key type to changed value audit log entry. The
// type can be found in the key's comment. Values can be nil.
//
// What
//
// I'm glad to see the same reaction that I had on you. In short, in this
// struct, the Key dictates what type NewValue and OldValue will have. They will
// always be the same type, but I will leave that as JSON for the user.
//
// Usage
//
// The usage of this is pretty simple, as AuditLogChange already has a
// convenient method to use. Here's an example on how to do "owner_id":
//
//    if change.Key != discord.AuditGuildOwnerID {
//        return errors.New("not owner ID")
//    }
//
//    // We know these are snowflakes because the comment said so for AuditGuildOwnerID.
//    var oldOwnerID, newOwnerID discord.Snowflake
//    if err := change.UnmarshalValues(&oldOwnerID, &newOwnerID); err != nil {
//        return err
//    }
//
//    log.Println("Transferred ownership from user", oldOwnerID, "to", newOwnerID)
//
type AuditLogChange struct {
	Key      string   `json:"key"`
	NewValue json.Raw `json:"new_value,omitempty"` // nullable
	OldValue json.Raw `json:"old_value,omitempty"` // nullable
}

func (a AuditLogChange) UnmarshalValues(old, new interface{}) error {
	if err := a.NewValue.UnmarshalTo(new); err != nil {
		return errors.Wrap(err, "Failed to unmarshal old value")
	}
	if err := a.OldValue.UnmarshalTo(old); err != nil {
		return errors.Wrap(err, "Failed to unmarshal new value")
	}
	return nil
}

type AuditLogChangeKey string

const (
	// Type string, name changed
	AuditGuildName AuditLogChangeKey = "name"
	// Type Hash, icon changed
	AuditGuildIconHash AuditLogChangeKey = "icon_hash"
	// Type Hash, invite splash page artwork changed
	AuditGuildSplashHash AuditLogChangeKey = "splash_hash"
	// Type Snowflake, owner changed
	AuditGuildOwnerID AuditLogChangeKey = "owner_id"
	// Type string, region changed
	AuditGuildRegion AuditLogChangeKey = "region"
	// Type Snowflake, afk channel changed
	AuditGuildAFKChannelID AuditLogChangeKey = "afk_channel_id"
	// Type Seconds, afk timeout duration changed
	AuditGuildAFKTimeout AuditLogChangeKey = "afk_timeout"
	// Type int, two-factor auth requirement changed
	AuditGuildMFA AuditLogChangeKey = "mfa_level"
	// Type Verification, required verification level changed
	AuditGuildVerification AuditLogChangeKey = "verification_level"
	// Type ExplicitFilter, change in whose messages are scanned and deleted for
	// explicit content in the server
	AuditGuildExplicitFilter AuditLogChangeKey = "explicit_content_filter"
	// Type Notification, default message notification level changed
	AuditGuildNotification AuditLogChangeKey = "default_message_notifications"
	// Type string, guild invite vanity URL changed
	AuditGuildVanityURLCode AuditLogChangeKey = "vanity_url_code"
	// Type []Role{ID, Name}, new role added
	AuditGuildRoleAdd AuditLogChangeKey = "$add"
	// Type []Role{ID, Name}, role removed
	AuditGuildRoleRemove AuditLogChangeKey = "$remove"
	// Type int, change in number of days after which inactive and
	// role-unassigned members are kicked
	AuditGuildPruneDeleteDays AuditLogChangeKey = "prune_delete_days"
	// Type bool, server widget enabled/disable
	AuditGuildWidgetEnabled AuditLogChangeKey = "widget_enabled"
	// Type Snowflake, channel ID of the server widget changed
	AuditGuildWidgetChannelID AuditLogChangeKey = "widget_channel_id"
	// Type Snowflake, ID of the system channel changed
	AuditGuildSystemChannelID AuditLogChangeKey = "system_channel_id"
)

const (
	// Type int, text or voice channel position changed
	AuditChannelPosition AuditLogChangeKey = "position"
	// Type string, text channel topic changed
	AuditChannelTopic AuditLogChangeKey = "topic"
	// Type uint, voice channel bitrate changed
	AuditChannelBitrate AuditLogChangeKey = "bitrate"
	// Type []Overwrite, permissions on a channel changed
	AuditChannelPermissionOverwrites AuditLogChangeKey = "permission_overwrites"
	// Type bool, channel NSFW restriction changed
	AuditChannelNSFW AuditLogChangeKey = "nsfw"
	// Type Snowflake, application ID of the added or removed webhook or bot
	AuditChannelApplicationID AuditLogChangeKey = "application_id"
	// Type Seconds, amount of seconds a user has to wait before sending another
	// message changed
	AuditChannelRateLimitPerUser AuditLogChangeKey = "rate_limit_per_user"
)

const (
	// Type Permissions, permissions for a role changed
	AuditRolePermissions AuditLogChangeKey = "permissions"
	// Type Color, role color changed
	AuditRoleColor AuditLogChangeKey = "color"
	// Type bool, role is now displayed/no longer displayed separate from online
	// users
	AuditRoleHoist AuditLogChangeKey = "hoist"
	// Type bool, role is now mentionable/unmentionable
	AuditRoleMentionable AuditLogChangeKey = "mentionable"
	// Type Permissions, a permission on a text or voice channel was allowed for
	// a role
	AuditRoleAllow AuditLogChangeKey = "allow"
	// Type Permissions, a permission on a text or voice channel was denied for
	// a role
	AuditRoleDeny AuditLogChangeKey = "deny"
)

const (
	// Type string, invite code changed
	AuditInviteCode AuditLogChangeKey = "code"
	// Type Snowflake, channel for invite code changed
	AuditInviteChannelID AuditLogChangeKey = "channel_id"
	// Type Snowflake, person who created invite code changed
	AuditInviteInviterID AuditLogChangeKey = "inviter_id"
	// Type int, change to max number of times invite code can be used
	AuditInviteMaxUses AuditLogChangeKey = "max_uses"
	// Type int, number of times invite code used changed
	AuditInviteUses AuditLogChangeKey = "uses"
	// Type Seconds, how long invite code lasts changed
	AuditInviteMaxAge AuditLogChangeKey = "max_age"
	// Type bool, invite code is temporary/never expires
	AuditInviteTemporary AuditLogChangeKey = "temporary"
)

const (
	// Type bool, user server deafened/undeafened
	AuditUserDeaf AuditLogChangeKey = "deaf"
	// Type bool, user server muted/unmuted
	AuditUserMute AuditLogChangeKey = "mute"
	// Type string, user nickname changed
	AuditUserNick AuditLogChangeKey = "nick"
	// Type Hash, user avatar changed
	AuditUserAvatarHash AuditLogChangeKey = "avatar_hash"
)

const (
	// Type Snowflake, the ID of the changed entity - sometimes used in
	// conjunction with other keys
	AuditAnyID AuditLogChangeKey = "id"
	// Type int (channel type) or string, type of entity created
	AuditAnyType AuditLogChangeKey = "type"
)

const (
	// Type bool, integration emoticons enabled/disabled
	AuditIntegrationEnableEmoticons AuditLogChangeKey = "enable_emoticons"
	// Type int, integration expiring subscriber behavior changed
	AuditIntegrationExpireBehavior AuditLogChangeKey = "expire_behavior"
	// Type int, integration expire grace period changed
	AuditIntegrationExpireGracePeriod AuditLogChangeKey = "expire_grace_period"
)
