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

func (NoopStore) Channel(discord.ChannelID) (*discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Channels(discord.GuildID) ([]discord.Channel, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) CreatePrivateChannel(discord.UserID) (*discord.Channel, error) {
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

func (NoopStore) Emoji(discord.GuildID, discord.EmojiID) (*discord.Emoji, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Emojis(discord.GuildID) ([]discord.Emoji, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) EmojiSet(discord.GuildID, []discord.Emoji) error {
	return nil
}

func (NoopStore) Guild(discord.GuildID) (*discord.Guild, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Guilds() ([]discord.Guild, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) GuildSet(*discord.Guild) error {
	return nil
}

func (NoopStore) GuildRemove(discord.GuildID) error {
	return nil
}

func (NoopStore) Member(discord.GuildID, discord.UserID) (*discord.Member, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Members(discord.GuildID) ([]discord.Member, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) MemberSet(discord.GuildID, *discord.Member) error {
	return nil
}

func (NoopStore) MemberRemove(discord.GuildID, discord.UserID) error {
	return nil
}

func (NoopStore) Message(discord.ChannelID, discord.MessageID) (*discord.Message, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Messages(discord.ChannelID) ([]discord.Message, error) {
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

func (NoopStore) MessageRemove(discord.ChannelID, discord.MessageID) error {
	return nil
}

func (NoopStore) Presence(discord.GuildID, discord.UserID) (*discord.Presence, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Presences(discord.GuildID) ([]discord.Presence, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) PresenceSet(discord.GuildID, *discord.Presence) error {
	return nil
}

func (NoopStore) PresenceRemove(discord.GuildID, discord.UserID) error {
	return nil
}

func (NoopStore) Role(discord.GuildID, discord.RoleID) (*discord.Role, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) Roles(discord.GuildID) ([]discord.Role, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) RoleSet(discord.GuildID, *discord.Role) error {
	return nil
}

func (NoopStore) RoleRemove(discord.GuildID, discord.RoleID) error {
	return nil
}

func (NoopStore) VoiceState(discord.GuildID, discord.UserID) (*discord.VoiceState, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) VoiceStates(discord.GuildID) ([]discord.VoiceState, error) {
	return nil, ErrNotImplemented
}

func (NoopStore) VoiceStateSet(discord.GuildID, *discord.VoiceState) error {
	return ErrNotImplemented
}

func (NoopStore) VoiceStateRemove(discord.GuildID, discord.UserID) error {
	return ErrNotImplemented
}
