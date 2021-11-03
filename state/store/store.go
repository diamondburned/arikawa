// Package store contains interfaces of the state's storage and its
// implementations.
//
// Getter Methods
//
// All getter methods will be wrapped by the State. If the State can't find
// anything in the storage, it will call the API itself and automatically add
// what's missing into the storage.
//
// Methods that return with a slice should pay attention to race conditions that
// would mutate the underlying slice (and as a result the returned slice as
// well). The best way to avoid this is to copy the whole slice, like
// defaultstore implementations do.
//
// Getter methods should not care about returning slices in order, unless
// explicitly stated against.
//
// ErrNotFound Rules
//
// If a getter method cannot find something, it should return ErrNotFound.
// Callers including State may check if the error is ErrNotFound to do something
// else. For example, if Guilds currently stores nothing, then it should return
// an empty slice and a nil error.
//
// In some cases, there may not be a way to know whether or not the store is
// unpopulated or is actually empty. In that case, implementations can return
// ErrNotFound when either happens. This will make State refetch from the API,
// so it is not ideal.
//
// Remove Methods
//
// Remove methods should return a nil error if the item it wants to delete is
// not found. This helps save some additional work in some cases.
package store

import (
	"errors"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

// ErrNotFound is an error that a store can use to return when something isn't
// in the storage. There is no strict restrictions on what uses this (the
// default one does, though), so be advised.
var ErrNotFound = errors.New("item not found in store")

// Cabinet combines all store interfaces into one but allows swapping individual
// stores out for another. Since the struct only consists of interfaces, it can
// be copied around.
type Cabinet struct {
	MeStore
	ChannelStore
	EmojiStore
	GuildStore
	MemberStore
	MessageStore
	PresenceStore
	RoleStore
	VoiceStateStore
}

// Reset resets everything inside the container.
func (sc *Cabinet) Reset() error {
	errors := []error{
		sc.MeStore.Reset(),
		sc.ChannelStore.Reset(),
		sc.EmojiStore.Reset(),
		sc.GuildStore.Reset(),
		sc.MemberStore.Reset(),
		sc.MessageStore.Reset(),
		sc.PresenceStore.Reset(),
		sc.RoleStore.Reset(),
		sc.VoiceStateStore.Reset(),
	}

	nonNils := errors[:0]

	for _, err := range errors {
		if err != nil {
			nonNils = append(nonNils, err)
		}
	}

	if len(nonNils) > 0 {
		return ResetErrors(nonNils)
	}

	return nil
}

// ResetErrors represents the multiple errors when StoreContainer is being
// resetted. A ResetErrors value must have at least 1 error.
type ResetErrors []error

// Error formats ResetErrors, showing the number of errors and the last error.
func (errs ResetErrors) Error() string {
	return fmt.Sprintf(
		"encountered %d reset errors (last: %v)",
		len(errs), errs[len(errs)-1],
	)
}

// Unwrap returns the last error in the list.
func (errs ResetErrors) Unwrap() error {
	return errs[len(errs)-1]
}

// append adds the error only if it is not nil.
func (errs *ResetErrors) append(err error) {
	if err != nil {
		*errs = append(*errs, err)
	}
}

// Noop is the value for a NoopStore.
var Noop = NoopStore{}

// NoopStore is a no-op implementation of all store interfaces. Its getters will
// always return ErrNotFound, and its setters will never return an error.
type NoopStore = noop

// NoopCabinet is a store cabinet with all store methods set to the Noop
// implementations.
var NoopCabinet = &Cabinet{
	MeStore:         Noop,
	ChannelStore:    Noop,
	EmojiStore:      Noop,
	GuildStore:      Noop,
	MemberStore:     Noop,
	MessageStore:    Noop,
	PresenceStore:   Noop,
	RoleStore:       Noop,
	VoiceStateStore: Noop,
}

// noop is the Noop type that implements methods.
type noop struct{}

// Resetter is an interface to reset the store on every Ready event.
type Resetter interface {
	// Reset resets the store to a new valid instance.
	Reset() error
}

type CoreStorer interface {
	Resetter
	Lock()
	Unlock()
}

var _ Resetter = (*noop)(nil)

func (noop) Reset() error { return nil }

// MeStore is the store interface for the current user.
type MeStore interface {
	Resetter

	Me() (*discord.User, error)
	MyselfSet(me discord.User, update bool) error
}

func (noop) Me() (*discord.User, error)         { return nil, ErrNotFound }
func (noop) MyselfSet(discord.User, bool) error { return nil }

// ChannelStore is the store interface for all channels.
type ChannelStore interface {
	Resetter

	// Channel searches for both DM and guild channels.
	Channel(discord.ChannelID) (*discord.Channel, error)
	// CreatePrivateChannel searches for private channels by the recipient ID.
	// It has the same API as *api.Client does.
	CreatePrivateChannel(recipient discord.UserID) (*discord.Channel, error)

	// Channels returns only channels from a guild.
	Channels(discord.GuildID) ([]discord.Channel, error)
	// PrivateChannels returns all private channels from the state.
	PrivateChannels() ([]discord.Channel, error)

	// Both ChannelSet and ChannelRemove should switch on Type to know if it's a
	// private channel or not.

	ChannelSet(c *discord.Channel, update bool) error
	ChannelRemove(*discord.Channel) error
}

var _ ChannelStore = (*noop)(nil)

func (noop) Channel(discord.ChannelID) (*discord.Channel, error) {
	return nil, ErrNotFound
}
func (noop) CreatePrivateChannel(discord.UserID) (*discord.Channel, error) {
	return nil, ErrNotFound
}
func (noop) Channels(discord.GuildID) ([]discord.Channel, error) {
	return nil, ErrNotFound
}
func (noop) PrivateChannels() ([]discord.Channel, error) {
	return nil, ErrNotFound
}
func (noop) ChannelSet(*discord.Channel, bool) error {
	return nil
}
func (noop) ChannelRemove(*discord.Channel) error {
	return nil
}

// EmojiStore is the store interface for all emojis.
type EmojiStore interface {
	Resetter

	Emoji(discord.GuildID, discord.EmojiID) (*discord.Emoji, error)
	Emojis(discord.GuildID) ([]discord.Emoji, error)

	// EmojiSet should delete all old emojis before setting new ones. The given
	// emojis slice will be a complete list of all emojis.
	EmojiSet(guildID discord.GuildID, emojis []discord.Emoji, update bool) error
}

var _ EmojiStore = (*noop)(nil)

func (noop) Emoji(discord.GuildID, discord.EmojiID) (*discord.Emoji, error) {
	return nil, ErrNotFound
}
func (noop) Emojis(discord.GuildID) ([]discord.Emoji, error) {
	return nil, ErrNotFound
}
func (noop) EmojiSet(discord.GuildID, []discord.Emoji, bool) error {
	return nil
}

// GuildStore is the store interface for all guilds.
type GuildStore interface {
	Resetter

	Guild(discord.GuildID) (*discord.Guild, error)
	Guilds() ([]discord.Guild, error)

	GuildSet(g *discord.Guild, update bool) error
	GuildRemove(id discord.GuildID) error
}

var _ GuildStore = (*noop)(nil)

func (noop) Guild(discord.GuildID) (*discord.Guild, error) { return nil, ErrNotFound }
func (noop) Guilds() ([]discord.Guild, error)              { return nil, ErrNotFound }
func (noop) GuildSet(*discord.Guild, bool) error           { return nil }
func (noop) GuildRemove(discord.GuildID) error             { return nil }

// MemberStore is the store interface for all members.
type MemberStore interface {
	Resetter

	Member(discord.GuildID, discord.UserID) (*discord.Member, error)
	Members(discord.GuildID) ([]discord.Member, error)

	MemberSet(guildID discord.GuildID, m *discord.Member, update bool) error
	MemberRemove(discord.GuildID, discord.UserID) error
}

var _ MemberStore = (*noop)(nil)

func (noop) Member(discord.GuildID, discord.UserID) (*discord.Member, error) {
	return nil, ErrNotFound
}
func (noop) Members(discord.GuildID) ([]discord.Member, error) {
	return nil, ErrNotFound
}
func (noop) MemberSet(discord.GuildID, *discord.Member, bool) error {
	return nil
}
func (noop) MemberRemove(discord.GuildID, discord.UserID) error {
	return nil
}

// MessageStore is the store interface for all messages.
type MessageStore interface {
	Resetter

	// MaxMessages returns the maximum number of messages. It is used to know if
	// the state cache is filled or not for one channel
	MaxMessages() int

	Message(discord.ChannelID, discord.MessageID) (*discord.Message, error)
	// Messages should return messages ordered from latest to earliest.
	Messages(discord.ChannelID) ([]discord.Message, error)

	// MessageSet either updates or adds a new message.
	//
	// A new message can be added, by setting update to false. Depending on
	// timestamp of the message, it will either be prepended or appended.
	//
	// If update is set to true, MessageSet will check if a message with the
	// id of the passed message is stored, and update it if so. Otherwise, if
	// there is no such message, it will be discarded.
	MessageSet(m *discord.Message, update bool) error
	MessageRemove(discord.ChannelID, discord.MessageID) error
}

var _ MessageStore = (*noop)(nil)

func (noop) MaxMessages() int {
	return 0
}
func (noop) Message(discord.ChannelID, discord.MessageID) (*discord.Message, error) {
	return nil, ErrNotFound
}
func (noop) Messages(discord.ChannelID) ([]discord.Message, error) {
	return nil, ErrNotFound
}
func (noop) MessageSet(*discord.Message, bool) error {
	return nil
}
func (noop) MessageRemove(discord.ChannelID, discord.MessageID) error {
	return nil
}

// PresenceStore is the store interface for all user presences. Presences don't get
// fetched from the API; they will only be updated through the Gateway.
type PresenceStore interface {
	Resetter

	Presence(discord.GuildID, discord.UserID) (*discord.Presence, error)
	Presences(discord.GuildID) ([]discord.Presence, error)

	PresenceSet(guildID discord.GuildID, p *discord.Presence, update bool) error
	PresenceRemove(discord.GuildID, discord.UserID) error
}

var _ PresenceStore = (*noop)(nil)

func (noop) Presence(discord.GuildID, discord.UserID) (*discord.Presence, error) {
	return nil, ErrNotFound
}
func (noop) Presences(discord.GuildID) ([]discord.Presence, error) {
	return nil, ErrNotFound
}
func (noop) PresenceSet(discord.GuildID, *discord.Presence, bool) error {
	return nil
}
func (noop) PresenceRemove(discord.GuildID, discord.UserID) error {
	return nil
}

// RoleStore is the store interface for all member roles.
type RoleStore interface {
	Resetter

	Role(discord.GuildID, discord.RoleID) (*discord.Role, error)
	Roles(discord.GuildID) ([]discord.Role, error)

	RoleSet(guildID discord.GuildID, r *discord.Role, update bool) error
	RoleRemove(discord.GuildID, discord.RoleID) error
}

var _ RoleStore = (*noop)(nil)

func (noop) Role(discord.GuildID, discord.RoleID) (*discord.Role, error) { return nil, ErrNotFound }
func (noop) Roles(discord.GuildID) ([]discord.Role, error)               { return nil, ErrNotFound }
func (noop) RoleSet(discord.GuildID, *discord.Role, bool) error          { return nil }
func (noop) RoleRemove(discord.GuildID, discord.RoleID) error            { return nil }

// VoiceStateStore is the store interface for all voice states.
type VoiceStateStore interface {
	Resetter

	VoiceState(discord.GuildID, discord.UserID) (*discord.VoiceState, error)
	VoiceStates(discord.GuildID) ([]discord.VoiceState, error)

	VoiceStateSet(guildID discord.GuildID, s *discord.VoiceState, update bool) error
	VoiceStateRemove(discord.GuildID, discord.UserID) error
}

var _ VoiceStateStore = (*noop)(nil)

func (noop) VoiceState(discord.GuildID, discord.UserID) (*discord.VoiceState, error) {
	return nil, ErrNotFound
}
func (noop) VoiceStates(discord.GuildID) ([]discord.VoiceState, error) {
	return nil, ErrNotFound
}
func (noop) VoiceStateSet(discord.GuildID, *discord.VoiceState, bool) error {
	return nil
}
func (noop) VoiceStateRemove(discord.GuildID, discord.UserID) error {
	return nil
}
