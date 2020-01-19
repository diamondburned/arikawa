package bot

/*
// UserPermissions but userID is after channelID.
func (ctx *Context) UserPermissions(channelID, userID string,
) (apermissions int, err error) {

	// Try to just get permissions from state.
	apermissions, err = ctx.Session.State.UserChannelPermissions(
		userID, channelID)
	if err == nil {
		return
	}

	// Otherwise try get as much data from state as possible, falling back to the network.
	channel, err := ctx.Channel(channelID)
	if err != nil {
		return
	}

	guild, err := ctx.Guild(channel.GuildID)
	if err != nil {
		return
	}

	if userID == guild.OwnerID {
		apermissions = discordgo.PermissionAll
		return
	}

	member, err := ctx.Member(guild.ID, userID)
	if err != nil {
		return
	}

	return MemberPermissions(guild, channel, member), nil
}

// Why this isn't exported, I have no idea.
func MemberPermissions(guild *discordgo.Guild, channel *discordgo.Channel,
	member *discordgo.Member) (apermissions int) {

	userID := member.User.ID

	if userID == guild.OwnerID {
		apermissions = discordgo.PermissionAll
		return
	}

	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			apermissions |= role.Permissions
			break
		}
	}

	for _, role := range guild.Roles {
		for _, roleID := range member.Roles {
			if role.ID == roleID {
				apermissions |= role.Permissions
				break
			}
		}
	}

	if apermissions&discordgo.PermissionAdministrator ==
		discordgo.PermissionAdministrator {

		apermissions |= discordgo.PermissionAll
	}

	// Apply @everyone overrides from the channel.
	for _, overwrite := range channel.PermissionOverwrites {
		if guild.ID == overwrite.ID {
			apermissions &= ^overwrite.Deny
			apermissions |= overwrite.Allow
			break
		}
	}

	denies := 0
	allows := 0

	// Member overwrites can override role overrides, so do two passes
	for _, overwrite := range channel.PermissionOverwrites {
		for _, roleID := range member.Roles {
			if overwrite.Type == "role" && roleID == overwrite.ID {
				denies |= overwrite.Deny
				allows |= overwrite.Allow
				break
			}
		}
	}

	apermissions &= ^denies
	apermissions |= allows

	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == "member" && overwrite.ID == userID {
			apermissions &= ^overwrite.Deny
			apermissions |= overwrite.Allow
			break
		}
	}

	if apermissions&discordgo.PermissionAdministrator ==
		discordgo.PermissionAdministrator {

		apermissions |= discordgo.PermissionAllChannel
	}

	return apermissions
}
*/
