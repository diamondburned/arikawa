package bot

import "testing"

func TestNewSubcommand(t *testing.T) {
	_, err := NewSubcommand(&testc{})
	if err != nil {
		t.Fatal("Failed to create new subcommand:", err)
	}
}

func TestSubcommand(t *testing.T) {
	var given = &testc{}
	var sub = &Subcommand{
		command: given,
	}

	t.Run("reflect commands", func(t *testing.T) {
		if err := sub.reflectCommands(); err != nil {
			t.Fatal("Failed to reflect commands:", err)
		}
	})

	t.Run("parse commands", func(t *testing.T) {
		if err := sub.parseCommands(); err != nil {
			t.Fatal("Failed to parse commands:", err)
		}

		// !!! CHANGE ME
		if len(sub.Commands) != 7 {
			t.Fatal("invalid ctx.commands len", len(sub.Commands))
		}

		var (
			foundSend   bool
			foundCustom bool
			foundNoArgs bool
		)

		for _, this := range sub.Commands {
			switch this.Command {
			case "send":
				foundSend = true
				if len(this.Arguments) != 1 {
					t.Fatal("invalid arguments len", len(this.Arguments))
				}

			case "custom":
				foundCustom = true
				if len(this.Arguments) != 1 {
					t.Fatal("arguments should be 1 for custom")
				}

			case "noArgs":
				foundNoArgs = true
				if len(this.Arguments) != 0 {
					t.Fatal("expected 0 arguments, got non-zero")
				}

			case "noop", "getCounter", "variadic", "trailCustom":
				// Found, but whatever
			}

			if this.event != typeMessageCreate {
				t.Fatal("invalid event type:", this.event.String())
			}
		}

		if !foundSend {
			t.Fatal("missing send")
		}

		if !foundCustom {
			t.Fatal("missing custom")
		}

		if !foundNoArgs {
			t.Fatal("missing noargs")
		}
	})

	t.Run("help commands", func(t *testing.T) {
		if h := sub.Help("", false); h == "" {
			t.Fatal("Empty subcommand help?")
		}
	})
}

func BenchmarkSubcommandConstructor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSubcommand(&testc{})
	}
}
