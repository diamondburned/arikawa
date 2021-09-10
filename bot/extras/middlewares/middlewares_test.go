package middlewares

import (
	"errors"
	"testing"

	"github.com/diamondburned/arikawa/v3/bot"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

func TestAdminOnly(t *testing.T) {
	var ctx = &bot.Context{
		State: &state.State{
			Session: &session.Session{
				Gateway: &gateway.Gateway{
					Identifier: &gateway.Identifier{
						IdentifyData: gateway.IdentifyData{
							Intents: option.NewUint(uint(gateway.IntentGuilds | gateway.IntentGuildMembers)),
						},
					},
				},
			},
			Cabinet: mockCabinet(),
		},
	}
	var middleware = AdminOnly(ctx)

	t.Run("allow message", func(t *testing.T) {
		var msg = &gateway.MessageCreateEvent{
			Message: discord.Message{
				ID:        1,
				ChannelID: 69420,
				Author:    discord.User{ID: 69420},
			},
		}
		expectNil(t, middleware(msg))
	})

	t.Run("deny message", func(t *testing.T) {
		var msg = &gateway.MessageCreateEvent{
			Message: discord.Message{
				ID:        2,
				ChannelID: 1337,
				Author:    discord.User{ID: 1337},
			},
		}
		expectBreak(t, middleware(msg))
		var pin = &gateway.ChannelPinsUpdateEvent{
			ChannelID: 120,
		}
		expectBreak(t, middleware(pin))
		var tpg = &gateway.TypingStartEvent{}
		expectBreak(t, middleware(tpg))
	})
}

func TestGuildOnly(t *testing.T) {
	var ctx = &bot.Context{
		State: &state.State{
			Session: &session.Session{
				Gateway: &gateway.Gateway{
					Identifier: &gateway.Identifier{
						IdentifyData: gateway.IdentifyData{
							Intents: option.NewUint(uint(gateway.IntentGuilds)),
						},
					},
				},
			},
			Cabinet: mockCabinet(),
		},
	}
	var middleware = GuildOnly(ctx)

	t.Run("allow message with GuildID", func(t *testing.T) {
		var msg = &gateway.MessageCreateEvent{
			Message: discord.Message{
				ID:      3,
				GuildID: 1337,
			},
		}
		expectNil(t, middleware(msg))
	})

	t.Run("allow message with ChannelID", func(t *testing.T) {
		var msg = &gateway.MessageCreateEvent{
			Message: discord.Message{
				ID:        3,
				ChannelID: 69420,
			},
		}
		expectNil(t, middleware(msg))
	})

	t.Run("deny message", func(t *testing.T) {
		var msg = &gateway.MessageCreateEvent{
			Message: discord.Message{
				ID:        1,
				ChannelID: 12,
			},
		}
		expectBreak(t, middleware(msg))

		var msg2 = &gateway.MessageCreateEvent{}
		expectBreak(t, middleware(msg2))
	})
}

func expectNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
}

func expectBreak(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, bot.Break) {
		return
	}
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	t.Fatal("Expected error, got nothing.")
}

// BenchmarkGuildOnly runs a message through the GuildOnly middleware to
// calculate the overhead of reflection.
func BenchmarkGuildOnly(b *testing.B) {
	var ctx = &bot.Context{
		State: &state.State{
			Cabinet: mockCabinet(),
		},
	}
	var middleware = GuildOnly(ctx)
	var msg = &gateway.MessageCreateEvent{
		Message: discord.Message{
			ID:      3,
			GuildID: 1337,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := middleware(msg); err != nil {
			b.Fatal("Unexpected error:", err)
		}
	}
}

// BenchmarkAdminOnly runs a message through the GuildOnly middleware to
// calculate the overhead of reflection.
func BenchmarkAdminOnly(b *testing.B) {
	var ctx = &bot.Context{
		State: &state.State{
			Cabinet: mockCabinet(),
		},
	}
	var middleware = AdminOnly(ctx)
	var msg = &gateway.MessageCreateEvent{
		Message: discord.Message{
			ID:        1,
			ChannelID: 1337,
			Author:    discord.User{ID: 69420},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := middleware(msg); err != nil {
			b.Fatal("Unexpected error:", err)
		}
	}
}

type mockStore struct {
	store.NoopStore
}

func mockCabinet() *store.Cabinet {
	c := *store.NoopCabinet
	c.GuildStore = &mockStore{}
	c.MemberStore = &mockStore{}
	c.ChannelStore = &mockStore{}

	return &c
}

func (s *mockStore) Guild(id discord.GuildID) (*discord.Guild, error) {
	return &discord.Guild{
		ID: id,
		Roles: []discord.Role{{
			ID:          69420,
			Permissions: discord.PermissionAdministrator,
		}},
	}, nil
}

func (s *mockStore) Member(_ discord.GuildID, userID discord.UserID) (*discord.Member, error) {
	return &discord.Member{
		User:    discord.User{ID: userID},
		RoleIDs: []discord.RoleID{discord.RoleID(userID)},
	}, nil
}

// Channel returns a channel with a guildID for #69420.
func (s *mockStore) Channel(id discord.ChannelID) (*discord.Channel, error) {
	if id == 69420 {
		return &discord.Channel{
			ID:      id,
			GuildID: 1337,
		}, nil
	}

	return &discord.Channel{
		ID: id,
	}, nil
}
