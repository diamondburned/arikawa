// +build unit

package bot

import "testing"

func TestNewSubcommand(t *testing.T) {
	_, err := NewSubcommand(&testCommands{})
	if err != nil {
		t.Fatal("Failed to create new subcommand:", err)
	}
}

func TestSubcommand(t *testing.T) {
	var given = &testCommands{}
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
		if len(sub.Commands) != 5 {
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
				if len(this.Arguments) > 0 {
					t.Fatal("arguments should be 0 for custom")
				}
				if this.parseType == nil {
					t.Fatal("custom has nil manualParse")
				}

			case "noargs":
				foundNoArgs = true
				if len(this.Arguments) != 0 {
					t.Fatal("expected 0 arguments, got non-zero")
				}
				if this.parseType != nil {
					t.Fatal("unexpected parseType")
				}

			case "noop", "getcounter":
				// Found, but whatever

			default:
				t.Fatal("Unexpected command:", this.Command)
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
}

func BenchmarkSubcommandConstructor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSubcommand(&testCommands{})
	}
}
