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

func (h *hasPlumb) Setup(sub *Subcommand) {
	sub.SetPlumb("Plumber")
}

func (h *hasPlumb) Normal(_ *gateway.MessageCreateEvent) error {
	h.NotPlumbed = true
	return nil
}

func (h *hasPlumb) Plumber(_ *gateway.MessageCreateEvent, c RawArguments) error {
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

	// Try call exactly what's in the Plumb example:
	m := &gateway.MessageCreateEvent{
		Message: discord.Message{
			Content: "hasPlumb",
		},
	}

	if err := c.callCmd(m); err != nil {
		t.Fatal("Failed to call message:", err)
	}

	if p.NotPlumbed {
		t.Fatal("Normal method called for hasPlumb")
	}
}

type onlyPlumb struct {
	Ctx     *Context
	Plumbed string
}

func (h *onlyPlumb) Setup(sub *Subcommand) {
	sub.SetPlumb("Plumber")
}

func (h *onlyPlumb) Plumber(_ *gateway.MessageCreateEvent, c RawArguments) error {
	h.Plumbed = string(c)
	return nil
}

func TestSubcommandOnlyPlumb(t *testing.T) {
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	c, err := New(state, &testc{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}
	c.HasPrefix = NewPrefix("")

	p := &onlyPlumb{}

	_, err = c.RegisterSubcommand(p)
	if err != nil {
		t.Fatal("Failed to register hasPlumb:", err)
	}

	// Try call exactly what's in the Plumb example:
	m := &gateway.MessageCreateEvent{
		Message: discord.Message{
			Content: "onlyPlumb test command",
		},
	}

	if err := c.callCmd(m); err != nil {
		t.Fatal("Failed to call message:", err)
	}

	if p.Plumbed != "test command" {
		t.Fatal("Unexpected custom argument for plumbed:", p.Plumbed)
	}
}
