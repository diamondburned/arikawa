package bot

import (
	"testing"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
)

type hasPlumb struct {
	Ctx *Context

	Plumbed    string
	NotPlumbed bool
}

func (h *hasPlumb) Normal(_ *gateway.MessageCreateEvent) error {
	h.NotPlumbed = true
	return nil
}

func (h *hasPlumb) PーPlumber(_ *gateway.MessageCreateEvent, c RawArguments) error {
	h.Plumbed = string(c)
	return nil
}

func TestSubcommandPlumb(t *testing.T) {
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	c, err := New(state, &testc{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}
	c.HasPrefix = NewPrefix("")

	p := &hasPlumb{}

	_, err = c.RegisterSubcommand(p)
	if err != nil {
		t.Fatal("Failed to register hasPlumb:", err)
	}

	if l := len(c.subcommands[0].Commands); l != 1 {
		t.Fatal("Unexpected length for sub.Commands:", l)
	}

	// Try call exactly what's in the Plumb example:
	m := &gateway.MessageCreateEvent{
		Message: discord.Message{
			Content: "hasPlumb test command",
		},
	}

	if err := c.callCmd(m); err != nil {
		t.Fatal("Failed to call message:", err)
	}

	if p.NotPlumbed {
		t.Fatal("Normal method called for hasPlumb")
	}

	if p.Plumbed != "hasPlumb test command" {
		t.Fatal("Unexpected custom argument for plumbed:", p.Plumbed)
	}
}
