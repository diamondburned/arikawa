package state

import (
	"errors"

	"github.com/diamondburned/arikawa/discord"
)

// Store is the state storage. It should handle mutex itself, and it should only
// concern itself with the local state.
type Store interface {
	StoreGetter
	StoreModifier
}

// All methods in StoreGetter will be wrapped by the State. If the State can't
// find anything in the storage, it will call the API itself and automatically
// add what's missing into the storage.
type StoreGetter interface {
	Self() (*discord.User, error)

	Channel(id discord.Snowflake) (*discord.Channel, error)
	Channels(guildID discord.Snowflake) ([]discord.Channel, error)
	PrivateChannels() ([]discord.Channel, error)

	Emoji(guildID, emojiID discord.Snowflake) (*discord.Emoji, error)
	Emojis(guildID discord.Snowflake) ([]discord.Emoji, error)

	Guild(id discord.Snowflake) (*discord.Guild, error)
	Guilds() ([]discord.Guild, error)

	Member(guildID, userID discord.Snowflake) (*discord.Member, error)
	Members(guildID discord.Snowflake) ([]discord.Member, error)

	Message(channelID, messageID discord.Snowflake) (*discord.Message, error)
	Messages(channelID discord.Snowflake) ([]discord.Message, error)

	Presence(guildID, userID discord.Snowflake) (*discord.Presence, error)
	Presences(guildID discord.Snowflake) ([]discord.Presence, error)

	Role(guildID, roleID discord.Snowflake) (*discord.Role, error)
	Roles(guildID discord.Snowflake) ([]discord.Role, error)
}

type StoreModifier interface {
	SelfSet(me *discord.User) error

	ChannelSet(*discord.Channel) error
	ChannelRemove(*discord.Channel) error

	EmojiSet(guildID discord.Snowflake, emojis []discord.Emoji) error

	GuildSet(*discord.Guild) error
	GuildRemove(*discord.Guild) error

	MemberSet(guildID discord.Snowflake, member *discord.Member) error
	MemberRemove(guildID, userID discord.Snowflake) error

	MessageSet(*discord.Message) error
	MessageRemove(*discord.Message) error

	PresenceSet(guildID discord.Snowflake, presence *discord.Presence) error
	PresenceRemove(guildID, userID discord.Snowflake) error

	RoleSet(guildID discord.Snowflake, role *discord.Role) error
	RoleRemove(guildID, roleID discord.Snowflake) error

	// This should reset all the state to zero/null.
	Reset() error
}

// ErrStoreNotFound is an error that a store can use to return when something
// isn't in the storage. There is no strict restrictions on what uses this (the
// default one does, though), so be advised.
var ErrStoreNotFound = errors.New("item not found in store")
