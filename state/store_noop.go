package state

import (
	"errors"

	"github.com/diamondburned/arikawa/discord"
)

// NoopStore could be embedded by other structs for partial state
// implementation. All Getters will return ErrNotImplemented, and all Setters
// will return no error.
type NoopStore struct{}

var _ Store = (*NoopStore)(nil)

var ErrNotImplemented = errors.New("state is not implemented")

func (NoopStore) Reset() error {
	return nil
}

func (NoopStore) Me() (*discord.User, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) MyselfSet(*discord.User) error {
	return nil
}

func (NoopStore) Channel(discord.Snowflake) (*discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Channels(discord.Snowflake) ([]discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) CreatePrivateChannel(discord.Snowflake) (*discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) PrivateChannels() ([]discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) ChannelSet(*discord.Channel) error {
	return nil
}

func (NoopStore) ChannelRemove(*discord.Channel) error {
	return nil
}

func (NoopStore) Emoji(_, _ discord.Snowflake) (*discord.Emoji, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Emojis(discord.Snowflake) ([]discord.Emoji, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) EmojiSet(discord.Snowflake, []discord.Emoji) error {
	return nil
}

func (NoopStore) Guild(discord.Snowflake) (*discord.Guild, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Guilds() ([]discord.Guild, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) GuildSet(*discord.Guild) error {
	return nil
}

func (NoopStore) GuildRemove(discord.Snowflake) error {
	return nil
}

func (NoopStore) Member(_, _ discord.Snowflake) (*discord.Member, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Members(discord.Snowflake) ([]discord.Member, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) MemberSet(discord.Snowflake, *discord.Member) error {
	return nil
}

func (NoopStore) MemberRemove(_, _ discord.Snowflake) error {
	return nil
}

func (NoopStore) Message(_, _ discord.Snowflake) (*discord.Message, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Messages(discord.Snowflake) ([]discord.Message, error) {
	return nil, ErrNotImplemented
}

// MaxMessages will always return 100 messages, so the API can fetch that
// many.
func (NoopStore) MaxMessages() int {
	return 100
}

func (NoopStore) MessageSet(*discord.Message) error {
	return nil
}

func (NoopStore) MessageRemove(_, _ discord.Snowflake) error {
	return nil
}

func (NoopStore) Presence(_, _ discord.Snowflake) (*discord.Presence, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Presences(discord.Snowflake) ([]discord.Presence, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) PresenceSet(discord.Snowflake, *discord.Presence) error {
	return nil
}

func (NoopStore) PresenceRemove(_, _ discord.Snowflake) error {
	return nil
}

func (NoopStore) Role(_, _ discord.Snowflake) (*discord.Role, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Roles(discord.Snowflake) ([]discord.Role, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) RoleSet(discord.Snowflake, *discord.Role) error {
	return nil
}

func (NoopStore) RoleRemove(_, _ discord.Snowflake) error {
	return nil
}

func (NoopStore) VoiceState(_, _ discord.Snowflake) (*discord.VoiceState, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) VoiceStates(_ discord.Snowflake) ([]discord.VoiceState, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) VoiceStateSet(discord.Snowflake, *discord.VoiceState) error {
	return ErrNotImplemented
}

func (NoopStore) VoiceStateRemove(_, _ discord.Snowflake) error {
	return ErrNotImplemented
}
