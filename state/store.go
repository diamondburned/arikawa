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
//
// Methods that return with a slice should pay attention to race conditions that
// would mutate the underlying slice (and as a result the returned slice as
// well). The best way to avoid this is to copy the whole slice, like
// DefaultStore does.
type StoreGetter interface {
	Me() (*discord.User, error)

	// Channel should check for both DM and guild channels.
	Channel(id discord.Snowflake) (*discord.Channel, error)
	Channels(guildID discord.Snowflake) ([]discord.Channel, error)

	// same API as (*api.Client)
	CreatePrivateChannel(recipient discord.Snowflake) (*discord.Channel, error)
	PrivateChannels() ([]discord.Channel, error)

	Emoji(guildID, emojiID discord.Snowflake) (*discord.Emoji, error)
	Emojis(guildID discord.Snowflake) ([]discord.Emoji, error)

	Guild(id discord.Snowflake) (*discord.Guild, error)
	Guilds() ([]discord.Guild, error)

	Member(guildID, userID discord.Snowflake) (*discord.Member, error)
	Members(guildID discord.Snowflake) ([]discord.Member, error)

	Message(channelID, messageID discord.Snowflake) (*discord.Message, error)
	// Messages should return messages ordered from latest to earliest.
	Messages(channelID discord.Snowflake) ([]discord.Message, error)
	MaxMessages() int // used to know if the state is filled or not.

	// These don't get fetched from the API, it's Gateway only.
	Presence(guildID, userID discord.Snowflake) (*discord.Presence, error)
	Presences(guildID discord.Snowflake) ([]discord.Presence, error)

	Role(guildID, roleID discord.Snowflake) (*discord.Role, error)
	Roles(guildID discord.Snowflake) ([]discord.Role, error)

	VoiceState(guildID discord.Snowflake, userID discord.Snowflake) (*discord.VoiceState, error)
	VoiceStates(guildID discord.Snowflake) ([]discord.VoiceState, error)
}

type StoreModifier interface {
	MyselfSet(me *discord.User) error

	// ChannelSet should switch on Type to know if it's a private channel or
	// not.
	ChannelSet(*discord.Channel) error
	ChannelRemove(*discord.Channel) error

	// EmojiSet should delete all old emojis before setting new ones.
	EmojiSet(guildID discord.Snowflake, emojis []discord.Emoji) error

	GuildSet(*discord.Guild) error
	GuildRemove(id discord.Snowflake) error

	MemberSet(guildID discord.Snowflake, member *discord.Member) error
	MemberRemove(guildID, userID discord.Snowflake) error

	// MessageSet should prepend messages into the slice, the latest being in
	// front.
	MessageSet(*discord.Message) error
	MessageRemove(channelID, messageID discord.Snowflake) error

	PresenceSet(guildID discord.Snowflake, presence *discord.Presence) error
	PresenceRemove(guildID, userID discord.Snowflake) error

	RoleSet(guildID discord.Snowflake, role *discord.Role) error
	RoleRemove(guildID, roleID discord.Snowflake) error

	VoiceStateSet(guildID discord.Snowflake, voiceState *discord.VoiceState) error
	VoiceStateRemove(guildID discord.Snowflake, userID discord.Snowflake) error
}

// ErrStoreNotFound is an error that a store can use to return when something
// isn't in the storage. There is no strict restrictions on what uses this (the
// default one does, though), so be advised.
var ErrStoreNotFound = errors.New("item not found in store")

// DiffMessage fills non-empty fields from src to dst.
func DiffMessage(src discord.Message, dst *discord.Message) {
	// Thanks, Discord.
	if src.Content != "" {
		dst.Content = src.Content
	}
	if src.EditedTimestamp.Valid() {
		dst.EditedTimestamp = src.EditedTimestamp
	}
	if src.Mentions != nil {
		dst.Mentions = src.Mentions
	}
	if src.Embeds != nil {
		dst.Embeds = src.Embeds
	}
	if src.Attachments != nil {
		dst.Attachments = src.Attachments
	}
	if src.Timestamp.Valid() {
		dst.Timestamp = src.Timestamp
	}
	if src.Author.ID.Valid() {
		dst.Author = src.Author
	}
	if src.Reactions != nil {
		dst.Reactions = src.Reactions
	}
}
