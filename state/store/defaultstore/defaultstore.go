// Package defaultstore provides thread-safe store implementations that store
// state values in memory.
package defaultstore

import "github.com/diamondburned/arikawa/v3/state/store"

// New creates a new cabinet instance of defaultstore. For Message, it creates a
// Message store with a limit of 100 messages.
func New() *store.Cabinet {
	return &store.Cabinet{
		MeStore:         NewMe(),
		ChannelStore:    NewChannel(),
		EmojiStore:      NewEmoji(),
		GuildStore:      NewGuild(),
		MemberStore:     NewMember(),
		MessageStore:    NewMessage(100),
		PresenceStore:   NewPresence(),
		RoleStore:       NewRole(),
		VoiceStateStore: NewVoiceState(),
	}
}
