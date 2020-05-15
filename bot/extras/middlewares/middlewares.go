package middlewares

import (
	"github.com/diamondburned/arikawa/bot"
	"github.com/diamondburned/arikawa/bot/extras/infer"
	"github.com/diamondburned/arikawa/discord"
)

func AdminOnly(ctx *bot.Context) func(interface{}) error {
	return func(ev interface{}) error {
		var channelID = infer.ChannelID(ev)
		if !channelID.Valid() {
			return bot.Break
		}

		var userID = infer.UserID(ev)
		if !userID.Valid() {
			return bot.Break
		}

		p, err := ctx.Permissions(channelID, userID)
		if err == nil && p.Has(discord.PermissionAdministrator) {
			return nil
		}

		return bot.Break
	}
}

func GuildOnly(ctx *bot.Context) func(interface{}) error {
	return func(ev interface{}) error {
		// Try and infer the GuildID.
		if guildID := infer.GuildID(ev); guildID.Valid() {
			return nil
		}

		var channelID = infer.ChannelID(ev)
		if !channelID.Valid() {
			return bot.Break
		}

		c, err := ctx.Channel(channelID)
		if err != nil || !c.GuildID.Valid() {
			return bot.Break
		}

		return nil
	}
}
