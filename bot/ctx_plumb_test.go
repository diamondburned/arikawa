package bot

import (
	"testing"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state"
	"github.com/diamondburned/arikawa/v2/state/store"
)

type hasPlumb struct {
	Ctx *Context

	Plumbed     bool
	PlumbedArgs string

	NotPlumbed     bool
	NotPlumbedArgs string
}

func (h *hasPlumb) Setup(sub *Subcommand) {
	sub.SetPlumb(h.Plumber)
}

func (h *hasPlumb) Plumber(_ *gateway.MessageCreateEvent, c RawArguments) error {
	h.NotPlumbed = false
	h.Plumbed = true
	h.PlumbedArgs = string(c)
	return nil
}

func (h *hasPlumb) Normal(_ *gateway.MessageCreateEvent, c RawArguments) error {
	h.Plumbed = false
	h.NotPlumbed = true
	h.NotPlumbedArgs = string(c)
	return nil
}

func TestSubcommandPlumb(t *testing.T) {
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	c, err := New(s, &testc{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}
	c.HasPrefix = NewPrefix("")

	p := &hasPlumb{}

	_, err = c.RegisterSubcommand(p)
	if err != nil {
		t.Fatal("Failed to register hasPlumb:", err)
	}

	sendFn := func(content string) {
		m := &gateway.MessageCreateEvent{
			Message: discord.Message{Content: content},
		}

		if err := c.callCmd(m); err != nil {
			t.Fatal("Failed to call message:", err)
		}
	}

	// Try call exactly what's in the Plumb example:
	sendFn("hasPlumb")

	if p.NotPlumbed || !p.Plumbed {
		t.Error("Normal method called for hasPlumb")
	}

	sendFn("hasPlumb arg1")

	if p.NotPlumbed || !p.Plumbed {
		t.Error("Normal method called for hasPlumb with arguments")
	}
	if p.PlumbedArgs != "arg1" {
		t.Errorf("Incorrect plumbed argument %q", p.PlumbedArgs)
	}

	sendFn("hasPlumb normal")

	if p.Plumbed || !p.NotPlumbed {
		t.Error("Plumbed method called for normal command")
	}

	sendFn("hasPlumb normal args")

	if p.Plumbed || !p.NotPlumbed {
		t.Error("Plumbed method called for normal command with arguments")
	}
	if p.NotPlumbedArgs != "args" {
		t.Errorf("Incorrect normal argument %q", p.NotPlumbedArgs)
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
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	c, err := New(s, &testc{})
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
