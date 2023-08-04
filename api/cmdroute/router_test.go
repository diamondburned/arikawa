package cmdroute

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"libdb.so/arikawa/v4/api"
	"libdb.so/arikawa/v4/discord"
	"libdb.so/arikawa/v4/utils/json"
	"libdb.so/arikawa/v4/utils/json/option"
)

func TestRouter(t *testing.T) {
	t.Run("command", func(t *testing.T) {
		r := NewRouter()
		r.Add("test", assertHandler(t, mockOptions))
		r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
			ID:      4,
			Name:    "test",
			Options: mockOptions,
		}))
	})

	t.Run("subcommand", func(t *testing.T) {
		r := NewRouter()
		r.Sub("test", func(r *Router) { r.Add("sub", assertHandler(t, mockOptions)) })
		r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
			ID:   4,
			Name: "test",
			Options: []discord.CommandInteractionOption{
				{
					Name:    "sub",
					Type:    discord.SubcommandOptionType,
					Options: mockOptions,
				},
			},
		}))
	})

	t.Run("unknown", func(t *testing.T) {
		r := NewRouter()
		r.AddFunc("test", func(ctx context.Context, data CommandData) *api.InteractionResponseData {
			t.Fatal("unexpected call")
			return nil
		})
		r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
			ID:   4,
			Name: "unknown",
		}))
	})

	t.Run("return", func(t *testing.T) {
		data := &api.InteractionResponseData{
			Content: option.NewNullableString("pong"),
		}

		r := NewRouter()
		r.AddFunc("ping", func(_ context.Context, _ CommandData) *api.InteractionResponseData {
			return data
		})
		resp := r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
			ID:      4,
			Name:    "ping",
			Options: mockOptions,
		}))

		if resp.Data != data {
			t.Fatal("unexpected response")
		}
	})

	t.Run("autocomplete", func(t *testing.T) {
		choices := []string{
			"foo",
			"bar",
			"baz",
		}

		r := NewRouter()
		r.AddFunc("ping", func(_ context.Context, _ CommandData) *api.InteractionResponseData {
			return nil
		})
		r.AddAutocompleterFunc("ping", func(_ context.Context, comp AutocompleteData) api.AutocompleteChoices {
			var data struct {
				Str string `discord:"str"`
			}

			if err := comp.Options.Unmarshal(&data); err != nil {
				t.Fatal("unexpected error:", err)
			}

			switch comp.Options.Focused().Name {
			case "str":
				matches := api.AutocompleteStringChoices{}
				for _, choice := range choices {
					if strings.HasPrefix(choice, data.Str) {
						matches = append(matches, discord.StringChoice{
							Name:  strings.ToUpper(choice),
							Value: choice,
						})
					}
				}
				return matches
			default:
				return nil
			}
		})

		assertInteractionResp(t,
			r.HandleInteraction(&discord.InteractionEvent{
				Token: "token",
				Data: &discord.AutocompleteInteraction{
					Name:        "ping",
					CommandType: discord.ChatInputCommand,
					Options: []discord.AutocompleteOption{
						{
							Type:    discord.StringOptionType,
							Name:    "str",
							Value:   json.Raw(`"b"`),
							Focused: true,
						},
					},
				},
			}),
			&api.InteractionResponse{
				Type: api.AutocompleteResult,
				Data: &api.InteractionResponseData{
					Choices: api.AutocompleteStringChoices{
						{Name: "BAR", Value: "bar"},
						{Name: "BAZ", Value: "baz"},
					},
				},
			},
		)
	})

	t.Run("component", func(t *testing.T) {
		r := NewRouter()
		r.AddComponentFunc("ping", func(ctx context.Context, data ComponentData) *api.InteractionResponse {
			button := data.ComponentInteraction.(*discord.ButtonInteraction)
			return &api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString(string(button.CustomID)),
				},
			}
		})
		resp := r.HandleInteraction(newInteractionEvent(&discord.ButtonInteraction{
			CustomID: "ping",
		}))
		if !reflect.DeepEqual(resp, &api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString("ping"),
			},
		}) {
			t.Fatal("unexpected response")
		}
	})

	t.Run("middlewares", func(t *testing.T) {
		var stack []string
		pushStack := func(s string) Middleware {
			return func(next InteractionHandler) InteractionHandler {
				return InteractionHandlerFunc(func(ctx context.Context, ev *discord.InteractionEvent) *api.InteractionResponse {
					stack = append(stack, s)
					return next.HandleInteraction(ctx, ev)
				})
			}
		}

		r := NewRouter()
		r.Use(pushStack("root1"))
		r.Use(pushStack("root2"))
		r.Sub("test", func(r *Router) {
			r.Use(pushStack("sub1.1"))
			r.Use(pushStack("sub1.2"))
			r.Sub("sub1", func(r *Router) {
				r.Use(pushStack("sub2.1"))
				r.Use(pushStack("sub2.2"))
				r.Add("sub2", assertHandler(t, mockOptions))
			})
		})
		r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
			ID:   4,
			Name: "test",
			Options: []discord.CommandInteractionOption{
				{
					Name: "sub1",
					Type: discord.SubcommandGroupOptionType,
					Options: []discord.CommandInteractionOption{
						{
							Name:    "sub2",
							Type:    discord.SubcommandOptionType,
							Options: mockOptions,
						},
					},
				},
			},
		}))

		expects := []string{
			"root1",
			"root2",
			"sub1.1",
			"sub1.2",
			"sub2.1",
			"sub2.2",
		}
		if len(stack) != len(expects) {
			t.Fatalf("expected stack to have %d elements, got %d", len(expects), len(stack))
		}

		for i := range expects {
			if stack[i] != expects[i] {
				t.Fatalf("expected stack[%d] to be %q, got %q", i, expects[i], stack[i])
			}
		}
	})

	t.Run("deferred", func(t *testing.T) {
		var wg sync.WaitGroup

		client := mockFollowUp(t, []followUpData{
			{
				token: "mock token",
				appID: 200,
				d: api.InteractionResponse{
					Type: api.MessageInteractionWithSource,
					Data: &api.InteractionResponseData{
						Content: option.NewNullableString("pong-defer"),
						Flags:   discord.EphemeralMessage,
					},
				},
			},
		})

		assertDeferred := func(t *testing.T, ctx context.Context, yes bool) {
			t.Helper()
			ticket := DeferTicketFromContext(ctx)
			if ticket.Context() == context.Background() {
				t.Error("expected ticket to be non-zero")
			}
			if ticket.IsDeferred() != yes {
				if yes {
					t.Error("expected ticket to not be deferred")
				} else {
					t.Error("expected ticket to be deferred")
				}
			}
		}

		r := NewRouter()
		r.Use(Deferrable(client, DeferOpts{
			Timeout: 100 * time.Millisecond,
			Flags:   discord.EphemeralMessage,
			Error:   func(err error) { t.Error(err) },
			Done:    func(*discord.Message) { wg.Done() },
		}))
		r.AddFunc("ping", func(ctx context.Context, data CommandData) *api.InteractionResponseData {
			assertDeferred(t, ctx, false)
			return &api.InteractionResponseData{
				Content: option.NewNullableString("pong"),
			}
		})
		r.AddFunc("ping-defer", func(ctx context.Context, data CommandData) *api.InteractionResponseData {
			assertDeferred(t, ctx, false)
			time.Sleep(200 * time.Millisecond)
			assertDeferred(t, ctx, true)
			return &api.InteractionResponseData{
				Content: option.NewNullableString("pong-defer"),
			}
		})

		assertInteractionResp(t,
			r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
				ID:      4,
				Name:    "ping",
				Options: mockOptions,
			})),
			&api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("pong"),
					Flags:   discord.EphemeralMessage,
				},
			},
		)

		wg.Add(1)
		assertInteractionResp(t,
			r.HandleInteraction(newInteractionEvent(&discord.CommandInteraction{
				ID:      4,
				Name:    "ping-defer",
				Options: mockOptions,
			})),
			&api.InteractionResponse{
				Type: api.DeferredMessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Flags: discord.EphemeralMessage,
				},
			},
		)

		wg.Wait()
	})
}

func newInteractionEvent(data discord.InteractionData) *discord.InteractionEvent {
	return &discord.InteractionEvent{
		ID:        100,
		AppID:     200,
		ChannelID: 300,
		Token:     "mock token",
		Data:      data,
	}
}

var mockOptions = []discord.CommandInteractionOption{
	{
		Name:  "value1",
		Type:  discord.NumberOptionType,
		Value: json.Raw("1"),
	},
	{
		Name:  "value2",
		Type:  discord.StringOptionType,
		Value: json.Raw("\"2\""),
	},
}

func assertHandler(t *testing.T, opts discord.CommandInteractionOptions) CommandHandler {
	return CommandHandlerFunc(func(ctx context.Context, data CommandData) *api.InteractionResponseData {
		if len(data.Options) != len(opts) {
			t.Fatalf("expected %d options, got %d", len(opts), len(data.Options))
		}

		for i, opt := range opts {
			if data.Options[i].Name != opt.Name {
				t.Fatalf("expected option %d to be %q, got %q", i, opt.Name, data.Options[i].Name)
			}

			if !bytes.Equal(data.Options[i].Value, opt.Value) {
				t.Fatalf("expected option %d to be %q, got %q", i, opt.Value, data.Options[i].Value)
			}
		}

		return nil
	})
}

type mockedFollowUpSender struct {
	t *testing.T
	d []followUpData
}

type followUpData struct {
	appID discord.AppID
	token string
	d     api.InteractionResponse
}

func mockFollowUp(t *testing.T, data []followUpData) *mockedFollowUpSender {
	return &mockedFollowUpSender{
		t: t,
		d: data,
	}
}

func (m *mockedFollowUpSender) FollowUpInteraction(appID discord.AppID, token string, d api.InteractionResponseData) (*discord.Message, error) {
	expect := m.d[0]
	m.d = m.d[1:]

	if appID != expect.appID {
		m.t.Errorf("expected appID to be %d, got %d", expect.appID, appID)
	}

	if token != expect.token {
		m.t.Errorf("expected token to be %q, got %q", expect.token, token)
	}

	if !reflect.DeepEqual(d, *expect.d.Data) {
		m.t.Errorf("unexpected interaction data\n"+
			"expected: %#v\n"+
			"got:      %#v", expect.d.Data, d)
	}

	return &discord.Message{}, nil
}

func assertInteractionResp(t *testing.T, got, expect *api.InteractionResponse) {
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("unexpected interaction\n"+
			"expected: %s\n"+
			"got:      %s",
			strInteractionResp(expect),
			strInteractionResp(got))
	}
}

func strInteractionResp(resp *api.InteractionResponse) string {
	if resp == nil {
		return "(*api.InteractionResponse)(nil)"
	}
	return fmt.Sprintf("%d:%#v", resp.Type, resp.Data)
}
