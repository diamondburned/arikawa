package discord

import (
	"time"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

// https://discord.com/developers/docs/resources/audit-log#audit-log-object
type AuditLog struct {
	// Webhooks is the list of webhooks found in the audit log.
	Webhooks []Webhook `json:"webhooks"`
	// Users is the list of users found in the audit log.
	Users []User `json:"users"`
	// Entries is the list of audit log entries.
	Entries []AuditLogEntry `json:"audit_log_entries"`
	// Integrations is a list ist of partial integration objects (only ID,
	// Name, Type, and Account).
	Integrations []Integration `json:"integrations"`
}

// AuditLogEntry is a single entry in the audit log.
//
// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object
type AuditLogEntry struct {
	// ID is the id of the entry.
	ID AuditLogEntryID `json:"id"`
	// TargetID is the id of the affected entity (webhook, user, role, etc.).
	TargetID Snowflake `json:"target_id"`
	// Changes are the changes made to the TargetID.
	Changes []AuditLogChange `json:"changes,omitempty"`
	// UserID is the id of the user who made the changes.
	UserID UserID `json:"user_id"`

	// ActionType is the type of action that occurred.
	ActionType AuditLogEvent `json:"action_type"`

	// Options contains additional info for certain action types.
	Options AuditEntryInfo `json:"options,omitempty"`
	// Reason is the reason for the change (0-512 characters).
	Reason string `json:"reason,omitempty"`
}

// CreatedAt returns a time object representing when the audit log entry was created.
func (e AuditLogEntry) CreatedAt() time.Time {
	return e.ID.Time()
}

// AuditLogEvent is the type of audit log action that occurred.
type AuditLogEvent uint8

// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-audit-log-events
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

// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-optional-audit-entry-info
type AuditEntryInfo struct {
	// DeleteMemberDays is the number of days after which inactive members were
	// kicked.
	//
	// Events: MEMBER_PRUNE
	DeleteMemberDays string `json:"delete_member_days,omitempty"`
	// MembersRemoved is the number of members removed by the prune.
	//
	// Events: MEMBER_PRUNE
	MembersRemoved string `json:"members_removed,omitempty"`
	// ChannelID is the id of the channel in which the entities were targeted.
	//
	// Events: MEMBER_MOVE, MESSAGE_PIN, MESSAGE_UNPIN, MESSAGE_DELETE
	ChannelID ChannelID `json:"channel_id,omitempty"`
	// MessagesID is the id of the message that was targeted.
	//
	// Events: MESSAGE_PIN, MESSAGE_UNPIN
	MessageID MessageID `json:"message_id,omitempty"`
	// Count is the number of entities that were targeted.
	//
	// Events: MESSAGE_DELETE, MESSAGE_BULK_DELETE, MEMBER_DISCONNECT,
	// MEMBER_MOVE
	Count string `json:"count,omitempty"`
	// ID is the id of the overwritten entity.
	//
	// Events: CHANNEL_OVERWRITE_CREATE, CHANNEL_OVERWRITE_UPDATE,
	// CHANNEL_OVERWRITE_DELETE
	ID Snowflake `json:"id,omitempty"`
	// Type is the type of overwritten entity.
	//
	// Events: CHANNEL_OVERWRITE_CREATE, CHANNEL_OVERWRITE_UPDATE,
	// CHANNEL_OVERWRITE_DELETE
	Type OverwriteType `json:"type,string,omitempty"`
	// RoleName is the name of the role if type is "role".
	//
	// Events: CHANNEL_OVERWRITE_CREATE, CHANNEL_OVERWRITE_UPDATE,
	// CHANNEL_OVERWRITE_DELETE
	RoleName string `json:"role_name,omitempty"`
}

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
//    // We know these are UserIDs because the comment said so for AuditGuildOwnerID.
//    var oldOwnerID, newOwnerID discord.UserID
//    if err := change.UnmarshalValues(&oldOwnerID, &newOwnerID); err != nil {
//        return err
//    }
//
//    log.Println("Transferred ownership from user", oldOwnerID, "to", newOwnerID)
//
type AuditLogChange struct {
	// Key is the name of audit log change key.
	Key AuditLogChangeKey `json:"key"`
	// NewValue is the new value of the key.
	NewValue json.Raw `json:"new_value,omitempty"`
	// OldValue is the old value of the key.
	OldValue json.Raw `json:"old_value,omitempty"`
}

// UnmarshalValues unmarshals the values of the AuditLogChange into the passed
// interfaces.
func (a AuditLogChange) UnmarshalValues(old, new interface{}) error {
	if err := a.NewValue.UnmarshalTo(new); err != nil {
		return errors.Wrap(err, "failed to unmarshal old value")
	}
	if err := a.OldValue.UnmarshalTo(old); err != nil {
		return errors.Wrap(err, "failed to unmarshal new value")
	}
	return nil
}

type AuditLogChangeKey string

// https://discord.com/developers/docs/resources/audit-log#audit-log-change-object-audit-log-change-key
const (
	// AuditGuildName gets sent if the guild's name was changed.
	//
	// Type: string
	AuditGuildName AuditLogChangeKey = "name"
	// AuditGuildIconHash gets sent if the guild's icon was changed.
	//
	// Type: Hash
	AuditGuildIconHash AuditLogChangeKey = "icon_hash"
	// AuditGuildSplashHash gets sent if the guild's invite splash page artwork
	// was changed.
	//
	// Type: Hash
	AuditGuildSplashHash AuditLogChangeKey = "splash_hash"
	// AuditGuildOwnerID gets sent if the guild's owner changed.
	//
	// Type: UserID
	AuditGuildOwnerID AuditLogChangeKey = "owner_id"
	// AuditGuildRegion gets sent if the guild's region changed.
	//
	// Type: string
	AuditGuildRegion AuditLogChangeKey = "region"
	// AuditGuildAFKChannelID gets sent if the guild's afk channel changed.
	//
	// Type: ChannelID
	AuditGuildAFKChannelID AuditLogChangeKey = "afk_channel_id"
	// AuditGuildAFKTimeout gets sent if the guild's afk timeout duration
	// changed.
	//
	// Type: Seconds
	AuditGuildAFKTimeout AuditLogChangeKey = "afk_timeout"
	// AuditGuildMFA gets sent if the two-factor auth requirement changed.
	//
	// Type: MFALevel
	AuditGuildMFA AuditLogChangeKey = "mfa_level"
	// AuditGuildVerification gets sent if the guild's required verification
	// level changed
	//
	// Type: Verification
	AuditGuildVerification AuditLogChangeKey = "verification_level"
	// AuditGuildExplicitFilter gets sent if there was a change in whose
	// messages are scanned and deleted for explicit content in the server.
	//
	// Type: ExplicitFilter
	AuditGuildExplicitFilter AuditLogChangeKey = "explicit_content_filter"
	// AuditGuildNotification gets sent if the default message notification
	// level changed.
	//
	// Type: Notification
	AuditGuildNotification AuditLogChangeKey = "default_message_notifications"
	// AuditGuildVanityURLCode gets sent if the guild invite vanity URL
	// changed.
	//
	// Type: string
	AuditGuildVanityURLCode AuditLogChangeKey = "vanity_url_code"
	// AuditGuildRoleAdd gets sent if a new role was added.
	//
	// Type: []Role{ID, Name}
	AuditGuildRoleAdd AuditLogChangeKey = "$add"
	// AuditGuildRoleRemove gets sent if a role was removed.
	//
	// Type: []Role{ID, Name}
	AuditGuildRoleRemove AuditLogChangeKey = "$remove"
	// AuditGuildPruneDeleteDays gets sent if there was a change in number of
	// days after which inactive and role-unassigned members are kicked.
	//
	// Type: int
	AuditGuildPruneDeleteDays AuditLogChangeKey = "prune_delete_days"
	// AuditGuildWidgetEnabled gets sent if the guild's widget was
	// enabled/disabled.
	//
	// Type: bool
	AuditGuildWidgetEnabled AuditLogChangeKey = "widget_enabled"
	// AuditGuildWidgetChannelID gets sent if the channel ID of the guild
	// widget changed.
	//
	// Type: ChannelID
	AuditGuildWidgetChannelID AuditLogChangeKey = "widget_channel_id"
	// AuditGuildSystemChannelID gets sent if the ID of the guild's system
	// channel changed.
	//
	// Type: ChannelID
	AuditGuildSystemChannelID AuditLogChangeKey = "system_channel_id"
)

const (
	// AuditChannelPosition gets sent if a text or voice channel position was
	// changed.
	//
	// Type: int
	AuditChannelPosition AuditLogChangeKey = "position"
	// AuditChannelTopic gets sent if the text channel topic changed.
	//
	// Type: string
	AuditChannelTopic AuditLogChangeKey = "topic"
	// AuditChannelBitrate gets sent if the voice channel bitrate changed.
	//
	// Type: uint
	AuditChannelBitrate AuditLogChangeKey = "bitrate"
	// AuditChannelPermissionOverwrites gets sent if the permissions on a
	// channel changed.
	//
	// Type: []Overwrite
	AuditChannelPermissionOverwrites AuditLogChangeKey = "permission_overwrites"
	// AuditChannelNSFW gets sent if the channel NSFW restriction changed.
	//
	// Type: bool
	AuditChannelNSFW AuditLogChangeKey = "nsfw"
	// AuditChannelApplicationID contains the application ID of the added or
	// removed webhook or bot.
	//
	// Type: AppID
	AuditChannelApplicationID AuditLogChangeKey = "application_id"
	// AuditChannelRateLimitPerUser gets sent if the amount of seconds a user
	// has to wait before sending another message changed.
	//
	// Type: Seconds
	AuditChannelRateLimitPerUser AuditLogChangeKey = "rate_limit_per_user"
)

const (
	// AuditRolePermissions gets sent if the permissions for a role changed.
	//
	// Type: Permissions
	AuditRolePermissions AuditLogChangeKey = "permissions"
	// AuditRoleColor gets sent if the role color changed.
	//
	// Type: Color
	AuditRoleColor AuditLogChangeKey = "color"
	// AuditRoleHoist gets sent if the role is now displayed/no longer
	// displayed separate from online users.
	//
	// Type: bool
	AuditRoleHoist AuditLogChangeKey = "hoist"
	// AuditRoleMentionable gets sent if a role is now
	// mentionable/unmentionable.
	//
	// Type: bool
	AuditRoleMentionable AuditLogChangeKey = "mentionable"
	// AuditRoleAllow gets sent if a permission on a text or voice channel was
	// allowed for a role.
	//
	// Type: Permissions
	AuditRoleAllow AuditLogChangeKey = "allow"
	// AuditRoleDeny gets sent if a permission on a text or voice channel was
	// denied for a role.
	//
	// Type: Permissions
	AuditRoleDeny AuditLogChangeKey = "deny"
)

const (
	// AuditInviteCode gets sent if an invite code changed.
	//
	// Type: string
	AuditInviteCode AuditLogChangeKey = "code"
	// AuditInviteChannelID gets sent if the channel for an invite code
	// changed.
	//
	// Type: ChannelID
	AuditInviteChannelID AuditLogChangeKey = "channel_id"
	// AuditInviteInviterID specifies the person who created invite code
	// changed.
	//
	// Type: UserID
	AuditInviteInviterID AuditLogChangeKey = "inviter_id"
	// AuditInviteMaxUses specifies the change to max number of times invite
	// code can be used.
	//
	// Type: int
	AuditInviteMaxUses AuditLogChangeKey = "max_uses"
	// AuditInviteUses specifies the number of times invite code used changed.
	//
	// Type: int
	AuditInviteUses AuditLogChangeKey = "uses"
	// AuditInviteMaxAge specifies the how long invite code lasts
	// changed.
	//
	// Type: Seconds
	AuditInviteMaxAge AuditLogChangeKey = "max_age"
	// AuditInviteTemporary specifies if an invite code is temporary/never
	// expires.
	//
	// Type: bool
	AuditInviteTemporary AuditLogChangeKey = "temporary"
)

const (
	// AuditUserDeaf specifies if the user was server deafened/undeafened.
	//
	// Type: bool
	AuditUserDeaf AuditLogChangeKey = "deaf"
	// AuditUserMute specifies if the user was server muted/unmuted.
	//
	// Type: bool
	AuditUserMute AuditLogChangeKey = "mute"
	// AuditUserNick specifies the new nickname of the user.
	//
	// Type: string
	AuditUserNick AuditLogChangeKey = "nick"
	// AuditUserAvatar specifies the hash of the new user avatar.
	//
	// Type: Hash
	AuditUserAvatarHash AuditLogChangeKey = "avatar_hash"
)

const (
	// AuditAnyID specifies the ID of the changed entity - sometimes used in
	// conjunction with other keys.
	//
	// Type: Snowflake
	AuditAnyID AuditLogChangeKey = "id"
	// AuditAnyType is the type of the entity created.
	// Type ChannelType or string
	AuditAnyType AuditLogChangeKey = "type"
)

const (
	// AuditIntegrationEnableEmoticons gets sent if the integration emoticons
	// were enabled/disabled.
	//
	// Type: bool
	AuditIntegrationEnableEmoticons AuditLogChangeKey = "enable_emoticons"
	// AuditIntegrationExpireBehavior gets sent if the integration expiring
	// subscriber behavior changed.
	//
	// Type: ExpireBehavior
	AuditIntegrationExpireBehavior AuditLogChangeKey = "expire_behavior"
	// AuditIntegrationExpireGracePeriod gets sent if the integration expire
	// grace period changed.
	//
	// Type: int
	AuditIntegrationExpireGracePeriod AuditLogChangeKey = "expire_grace_period"
)
