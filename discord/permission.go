package discord

type Permissions uint64

var (
	// Allows creation of instant invites
	PermissionCreateInstantInvite Permissions = 1 << 0
	// Allows kicking members
	PermissionKickMembers Permissions = 1 << 1
	// Allows banning members
	PermissionBanMembers Permissions = 1 << 2
	// Allows all permissions and bypasses channel permission overwrites
	PermissionAdministrator Permissions = 1 << 3
	// Allows management and editing of channels
	PermissionManageChannels Permissions = 1 << 4
	// Allows management and editing of the guild
	PermissionManageGuild Permissions = 1 << 5
	// Allows for the addition of reactions to messages
	PermissionAddReactions Permissions = 1 << 6
	// Allows for viewing of audit logs
	PermissionViewAuditLog Permissions = 1 << 7
	// Allows for using priority speaker in a voice channel
	PermissionPrioritySpeaker Permissions = 1 << 8
	// Allows the user to go live
	PermissionStream Permissions = 1 << 9
	// Allows guild members to view a channel, which includes reading messages
	// in text channels
	PermissionViewChannel Permissions = 1 << 10
	// Allows for sending messages in a channel
	PermissionSendMessages Permissions = 1 << 11
	// Allows for sending of /tts messages
	PermissionSendTTSMessages Permissions = 1 << 12
	// Allows for deletion of other users messages
	PermissionManageMessages Permissions = 1 << 13
	// Links sent by users with this permission will be auto-embedded
	PermissionEmbedLinks Permissions = 1 << 14
	// Allows for uploading images and files
	PermissionAttachFiles Permissions = 1 << 15
	// Allows for reading of message history
	PermissionReadMessageHistory Permissions = 1 << 16
	// Allows for using the @everyone tag to notify all users in a channel,
	// and the @here tag to notify all online users in a channel
	PermissionMentionEveryone Permissions = 1 << 17
	// Allows the usage of custom emojis from other servers
	PermissionUseExternalEmojis Permissions = 1 << 18

	// ?

	// Allows for joining of a voice channel
	PermissionConnect Permissions = 1 << 20
	// Allows for speaking in a voice channel
	PermissionSpeak Permissions = 1 << 21
	// Allows for muting members in a voice channel
	PermissionMuteMembers Permissions = 1 << 22
	// Allows for deafening of members in a voice channel
	PermissionDeafenMembers Permissions = 1 << 23
	// Allows for moving of members between voice channels
	PermissionMoveMembers Permissions = 1 << 24
	// Allows for using voice-activity-detection in a voice channel
	PermissionUseVAD Permissions = 1 << 25
	// Allows for modification of own nickname
	PermissionChangeNickname Permissions = 1 << 26
	// Allows for modification of other users nicknames
	PermissionManageNicknames Permissions = 1 << 27
	// Allows management and editing of roles
	PermissionManageRoles Permissions = 1 << 28
	// Allows management and editing of webhooks
	PermissionManageWebhooks Permissions = 1 << 29
	// Allows management and editing of emojis
	PermissionManageEmojis Permissions = 1 << 30

	PermissionAllText = 0 |
		PermissionViewChannel |
		PermissionSendMessages |
		PermissionSendTTSMessages |
		PermissionManageMessages |
		PermissionEmbedLinks |
		PermissionAttachFiles |
		PermissionReadMessageHistory |
		PermissionMentionEveryone |
		PermissionUseExternalEmojis

	PermissionAllVoice = 0 |
		PermissionConnect |
		PermissionSpeak |
		PermissionMuteMembers |
		PermissionDeafenMembers |
		PermissionMoveMembers |
		PermissionUseVAD |
		PermissionPrioritySpeaker

	PermissionAllChannel = 0 |
		PermissionAllText |
		PermissionAllVoice |
		PermissionCreateInstantInvite |
		PermissionManageRoles |
		PermissionManageChannels |
		PermissionAddReactions |
		PermissionViewAuditLog

	PermissionAll = 0 |
		PermissionAllChannel |
		PermissionKickMembers |
		PermissionBanMembers |
		PermissionManageGuild |
		PermissionAdministrator |
		PermissionManageWebhooks |
		PermissionManageEmojis |
		PermissionManageNicknames |
		PermissionChangeNickname
)

func (p Permissions) Has(perm Permissions) bool {
	return HasFlag(uint64(p), uint64(perm))
}

func (p Permissions) Add(perm Permissions) Permissions {
	return p | perm
}

func CalcOverwrites(guild Guild, channel Channel, member Member) Permissions {
	if guild.OwnerID == member.User.ID {
		return PermissionAll
	}

	var perm Permissions

	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			perm |= role.Permissions
			break
		}
	}

	for _, role := range guild.Roles {
		for _, id := range member.RoleIDs {
			if id == role.ID {
				perm |= role.Permissions
				break
			}
		}
	}

	if perm.Has(PermissionAdministrator) {
		return PermissionAll
	}

	for _, overwrite := range channel.Permissions {
		if overwrite.ID == guild.ID {
			perm &= ^overwrite.Deny
			perm |= overwrite.Allow
			break
		}
	}

	var deny, allow Permissions

	for _, overwrite := range channel.Permissions {
		for _, id := range member.RoleIDs {
			if id == overwrite.ID && overwrite.Type == "role" {
				deny |= overwrite.Deny
				allow |= overwrite.Allow
				break
			}
		}
	}

	perm &= ^deny
	perm |= allow

	for _, overwrite := range channel.Permissions {
		if overwrite.ID == member.User.ID {
			perm &= ^overwrite.Deny
			perm |= overwrite.Allow
			break
		}
	}

	if perm.Has(PermissionAdministrator) {
		return PermissionAll
	}

	return perm
}
