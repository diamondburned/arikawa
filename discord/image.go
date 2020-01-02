package discord

type Image string

const (
	ImageBaseURL = "https://cdn.discordapp.com/"
)

type ImageType uint8

const (
	CustomEmoji ImageType = iota
	GuildIcon
	GuildSplash
	GuildBanner
	DefaultUserAvatar
	UserAvatar
	ApplicationIcon
	ApplicationAsset
	AchievementIcon
	TeamIcon
)
