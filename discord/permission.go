package discord

type Permissions uint64

// https://discord.com/developers/docs/topics/permissions#permissions-bitwise-permission-flags
const (
	// Allows creation of instant invites
	PermissionCreateInstantInvite Permissions = 1 << iota
	// Allows kicking members
	PermissionKickMembers
	// Allows banning members
	PermissionBanMembers
	// Allows all permissions and bypasses channel permission overwrites
	PermissionAdministrator
	// Allows management and editing of channels
	PermissionManageChannels
	// Allows management and editing of the guild
	PermissionManageGuild
	// Allows for the addition of reactions to messages
	PermissionAddReactions
	// Allows for viewing of audit logs
	PermissionViewAuditLog
	// Allows for using priority speaker in a voice channel
	PermissionPrioritySpeaker
	// Allows the user to go live
	PermissionStream
	// Allows guild members to view a channel, which includes reading messages
	// in text channels
	PermissionViewChannel
	// Allows for sending messages in a channel
	PermissionSendMessages
	// Allows for sending of /tts messages
	PermissionSendTTSMessages
	// Allows for deletion of other users messages
	PermissionManageMessages
	// Links sent by users with this permission will be auto-embedded
	PermissionEmbedLinks
	// Allows for uploading images and files
	PermissionAttachFiles
	// Allows for reading of message history
	PermissionReadMessageHistory
	// Allows for using the @everyone tag to notify all users in a channel,
	// and the @here tag to notify all online users in a channel
	PermissionMentionEveryone
	// Allows the usage of custom emojis from other servers
	PermissionUseExternalEmojis
	// ?
	_
	// Allows for joining of a voice channel
	PermissionConnect
	// Allows for speaking in a voice channel
	PermissionSpeak
	// Allows for muting members in a voice channel
	PermissionMuteMembers
	// Allows for deafening of members in a voice channel
	PermissionDeafenMembers
	// Allows for moving of members between voice channels
	PermissionMoveMembers
	// Allows for using voice-activity-detection in a voice channel
	PermissionUseVAD
	// Allows for modification of own nickname
	PermissionChangeNickname
	// Allows for modification of other users nicknames
	PermissionManageNicknames
	// Allows management and editing of roles
	PermissionManageRoles
	// Allows management and editing of webhooks
	PermissionManageWebhooks
	// Allows members to use slash commands in text channels
	PermissionManageEmojisAndStickers
	// Allows members to use slash commands in text channels
	PermissionUseSlashCommands
	// Allows for requesting to speak in stage channels. (This permission is
	// under active development and may be changed or removed.)
	PermissionRequestToSpeak
	_
	// Allows for deleting and archiving threads, and viewing all private
	// threads
	PermissionManageThreads
	// Allows for creating and participating in threads.
	PermissionUsePublicThreads
	// Allows for creating and participating in private threads.
	PermissionUsePrivateThreads
	// Allows the usage of custom stickers from other servers
	PermissionUseExternalStickers

	PermissionAllText = 0 |
		PermissionViewChannel |
		PermissionSendMessages |
		PermissionSendTTSMessages |
		PermissionManageMessages |
		PermissionEmbedLinks |
		PermissionAttachFiles |
		PermissionReadMessageHistory |
		PermissionMentionEveryone |
		PermissionUseExternalEmojis |
		PermissionUseSlashCommands |
		PermissionUsePublicThreads |
		PermissionUsePrivateThreads |
		PermissionUseExternalStickers

	PermissionAllVoice = 0 |
		PermissionViewChannel |
		PermissionConnect |
		PermissionSpeak |
		PermissionStream |
		PermissionMuteMembers |
		PermissionDeafenMembers |
		PermissionMoveMembers |
		PermissionUseVAD |
		PermissionPrioritySpeaker |
		PermissionRequestToSpeak

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
		PermissionManageEmojisAndStickers |
		PermissionManageNicknames |
		PermissionChangeNickname |
		PermissionManageThreads
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
		if role.ID == RoleID(guild.ID) {
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

	for _, overwrite := range channel.Overwrites {
		if GuildID(overwrite.ID) == guild.ID {
			perm &= ^overwrite.Deny
			perm |= overwrite.Allow
			break
		}
	}

	var deny, allow Permissions

	for _, overwrite := range channel.Overwrites {
		for _, id := range member.RoleIDs {
			if id == RoleID(overwrite.ID) && overwrite.Type == OverwriteRole {
				deny |= overwrite.Deny
				allow |= overwrite.Allow
				break
			}
		}
	}

	perm &= ^deny
	perm |= allow

	for _, overwrite := range channel.Overwrites {
		if UserID(overwrite.ID) == member.User.ID {
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
