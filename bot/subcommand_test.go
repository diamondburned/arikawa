package bot

import (
	"strings"
	"testing"
)

func TestUnderline(t *testing.T) {
	HelpUnderline = false
	if underline("astolfo") != "astolfo" {
		t.Fatal("Unexpected underlining with HelpUnderline = false")
	}

	HelpUnderline = true
	if underline("arikawa hime") != "__arikawa hime__" {
		t.Fatal("Unexpected normal style with HelpUnderline = true")
	}
}

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
		if len(sub.Commands) < 8 {
			t.Fatal("too low sub.Methods len", len(sub.Commands))
		}
		if len(sub.Events) < 1 {
			t.Fatal("No events found.")
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

	t.Run("init commands", func(t *testing.T) {
		ctx := &Context{}
		if err := sub.InitCommands(ctx); err != nil {
			t.Fatal("Failed to init commands:", err)
		}
	})

	t.Run("help commands", func(t *testing.T) {
		h := sub.Help()
		if h == "" {
			t.Fatal("Empty subcommand help?")
		}

		if strings.Contains(h, "hidden") {
			t.Fatal("Hidden command shown in help:\n", h)
		}
	})

	t.Run("change command", func(t *testing.T) {
		sub.ChangeCommandInfo("Noop", "crossdressing", "best")
		if h := sub.Help(); !strings.Contains(h, "crossdressing: best") {
			t.Fatal("Changed command is not in help.")
		}
	})
}

func BenchmarkSubcommandConstructor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSubcommand(&testc{})
	}
}
