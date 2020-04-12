// +build unit

package bot

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
)

type testCommands struct {
	Ctx     *Context
	Return  chan interface{}
	Counter uint64
	Typed   bool
}

func (t *testCommands) MーBumpCounter(interface{}) error {
	t.Counter++
	return nil
}

func (t *testCommands) GetCounter(_ *gateway.MessageCreateEvent) error {
	t.Return <- strconv.FormatUint(t.Counter, 10)
	return nil
}

func (t *testCommands) Send(_ *gateway.MessageCreateEvent, arg string) error {
	t.Return <- arg
	return errors.New("oh no")
}

func (t *testCommands) Custom(_ *gateway.MessageCreateEvent, c *customParseable) error {
	t.Return <- c.args
	return nil
}

func (t *testCommands) NoArgs(_ *gateway.MessageCreateEvent) error {
	return errors.New("passed")
}

func (t *testCommands) Noop(_ *gateway.MessageCreateEvent) error {
	return nil
}

func (t *testCommands) OnTyping(_ *gateway.TypingStartEvent) error {
	t.Typed = true
	return nil
}

type customParseable struct {
	args []string
}

func (c *customParseable) ParseContent(args []string) error {
	c.args = args
	return nil
}

func TestNewContext(t *testing.T) {
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	c, err := New(state, &testCommands{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}

	if !reflect.DeepEqual(c.Subcommands(), c.subcommands) {
		t.Fatal("Subcommands mismatch.")
	}
}

func TestContext(t *testing.T) {
	var given = &testCommands{}
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	s, err := NewSubcommand(given)
	if err != nil {
		t.Fatal("Failed to create subcommand:", err)
	}

	var ctx = &Context{
		Subcommand: s,
		State:      state,
	}

	t.Run("init commands", func(t *testing.T) {
		if err := ctx.Subcommand.InitCommands(ctx); err != nil {
			t.Fatal("Failed to init commands:", err)
		}

		if given.Ctx == nil {
			t.Fatal("given's Context field is nil")
		}

		if given.Ctx.State.Store == nil {
			t.Fatal("given's State is nil")
		}
	})

	t.Run("find commands", func(t *testing.T) {
		cmd := ctx.FindCommand("", "NoArgs")
		if cmd == nil {
			t.Fatal("Failed to find NoArgs")
		}
	})

	t.Run("help", func(t *testing.T) {
		if h := ctx.Help(); h == "" {
			t.Fatal("Empty help?")
		}
		if h := ctx.HelpAdmin(); h == "" {
			t.Fatal("Empty admin help?")
		}
	})

	testReturn := func(expects interface{}, content string) (call error) {
		// Return channel for testing
		ret := make(chan interface{})
		given.Return = ret

		// Mock a messageCreate event
		m := &gateway.MessageCreateEvent{
			Message: discord.Message{
				Content: content,
			},
		}

		var (
			callCh = make(chan error)
		)

		go func() {
			callCh <- ctx.callCmd(m)
		}()

		select {
		case arg := <-ret:
			if !reflect.DeepEqual(arg, expects) {
				t.Fatal("returned argument is invalid:", arg)
			}
			call = <-callCh

		case call = <-callCh:
			t.Fatal("expected return before error:", call)
		}

		return
	}

	t.Run("middleware", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("pls do ")

		// This should trigger the middleware first.
		if err := testReturn("1", "pls do getCounter"); err != nil {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("typing event", func(t *testing.T) {
		typing := &gateway.TypingStartEvent{}

		if err := ctx.callCmd(typing); err != nil {
			t.Fatal("Failed to call with TypingStart:", err)
		}

		if !given.Typed {
			t.Fatal("Typed bool is false")
		}
	})

	t.Run("call command", func(t *testing.T) {
		// Set a custom prefix
		ctx.HasPrefix = NewPrefix("~")

		if err := testReturn("test", "~send test"); err.Error() != "oh no" {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("call command custom parser", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("!")
		expects := []string{"custom", "arg1", ":)"}

		if err := testReturn(expects, "!custom arg1 :)"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}
	})

	testMessage := func(content string) error {
		// Mock a messageCreate event
		m := &gateway.MessageCreateEvent{
			Message: discord.Message{
				Content: content,
			},
		}

		return ctx.callCmd(m)
	}

	t.Run("call command without args", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("")

		if err := testMessage("noArgs"); err.Error() != "passed" {
			t.Fatal("unexpected error:", err)
		}
	})

	// Test error cases

	t.Run("call unknown command", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("joe pls ")

		err := testMessage("joe pls no")

		if err == nil || !strings.HasPrefix(err.Error(), "Unknown command:") {
			t.Fatal("unexpected error:", err)
		}
	})

	// Test subcommands

	t.Run("register subcommand", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("run ")

		_, err := ctx.RegisterSubcommand(&testCommands{})
		if err != nil {
			t.Fatal("Failed to register subcommand:", err)
		}

		if err := testMessage("run testCommands noop"); err != nil {
			t.Fatal("Unexpected error:", err)
		}

		cmd := ctx.FindCommand("testCommands", "Noop")
		if cmd == nil {
			t.Fatal("Failed to find subcommand Noop")
		}
	})
}

func BenchmarkConstructor(b *testing.B) {
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	for i := 0; i < b.N; i++ {
		_, _ = New(state, &testCommands{})
	}
}

func BenchmarkCall(b *testing.B) {
	var given = &testCommands{}
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	s, _ := NewSubcommand(given)

	var ctx = &Context{
		Subcommand: s,
		State:      state,
		HasPrefix:  NewPrefix("~"),
	}

	m := &gateway.MessageCreateEvent{
		Message: discord.Message{
			Content: "~noop",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.callCmd(m)
	}
}

func BenchmarkHelp(b *testing.B) {
	var given = &testCommands{}
	var state = &state.State{
		Store: state.NewDefaultStore(nil),
	}

	s, _ := NewSubcommand(given)

	var ctx = &Context{
		Subcommand: s,
		State:      state,
		HasPrefix:  NewPrefix("~"),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.Help()
	}
}

type hasID struct {
	ChannelID discord.Snowflake
}

type embedsID struct {
	*hasID
	*embedsID
}

type hasChannelInName struct {
	ID discord.Snowflake
}

func TestReflectChannelID(t *testing.T) {
	var s = &hasID{
		ChannelID: 69420,
	}

	t.Run("hasID", func(t *testing.T) {
		if id := reflectChannelID(s); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("embedsID", func(t *testing.T) {
		var e = &embedsID{
			hasID: s,
		}

		if id := reflectChannelID(e); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("hasChannelInName", func(t *testing.T) {
		var s = &hasChannelInName{
			ID: 69420,
		}

		if id := reflectChannelID(s); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})
}

func BenchmarkReflectChannelID_1Level(b *testing.B) {
	var s = &hasID{
		ChannelID: 69420,
	}

	for i := 0; i < b.N; i++ {
		_ = reflectChannelID(s)
	}
}

func BenchmarkReflectChannelID_5Level(b *testing.B) {
	var s = &embedsID{
		nil,
		&embedsID{
			nil,
			&embedsID{
				nil,
				&embedsID{
					hasID: &hasID{
						ChannelID: 69420,
					},
				},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = reflectChannelID(s)
	}
}
