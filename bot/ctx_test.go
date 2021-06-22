package bot

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state"
	"github.com/diamondburned/arikawa/v2/state/store"
	"github.com/diamondburned/arikawa/v2/utils/handler"
)

type testUnexportedCtx struct {
	ctx *Context
}

func TestUnexportedCtx(t *testing.T) {
	s := &state.State{
		Cabinet: store.NoopCabinet,
	}

	_, err := New(s, &testUnexportedCtx{})
	if err == nil {
		t.Fatal("New returned unexpected nil error")
	}

	if !strings.Contains(err.Error(), "no exported field with *bot.Context found") {
		t.Fatal("unexpected New error:", err)
	}
}

type testc struct {
	Ctx     *Context
	Return  chan interface{}
	Counter uint64
	Typed   int8
}

func (t *testc) Setup(sub *Subcommand) {
	sub.AddMiddleware([]string{"*", "GetCounter"}, func(v interface{}) {
		t.Counter++
	})
	sub.AddMiddleware("*", func(*gateway.MessageCreateEvent) {
		t.Counter++
	})
	// stub middleware for testing
	sub.AddMiddleware(t.OnTyping, func(*gateway.TypingStartEvent) {
		t.Typed = 2
	})
	sub.Hide(t.Hidden)
}
func (t *testc) Hidden(*gateway.MessageCreateEvent) {}
func (t *testc) Noop(*gateway.MessageCreateEvent)   {}
func (t *testc) GetCounter(*gateway.MessageCreateEvent) {
	t.Return <- strconv.FormatUint(t.Counter, 10)
}
func (t *testc) Send(_ *gateway.MessageCreateEvent, args ...string) error {
	t.Return <- args
	return errors.New("oh no")
}
func (t *testc) Custom(_ *gateway.MessageCreateEvent, c *ArgumentParts) {
	t.Return <- []string(*c)
}
func (t *testc) Variadic(_ *gateway.MessageCreateEvent, c ...*customParsed) {
	t.Return <- c[len(c)-1]
}
func (t *testc) TrailCustom(_ *gateway.MessageCreateEvent, _ string, c ArgumentParts) {
	t.Return <- c
}
func (t *testc) Content(_ *gateway.MessageCreateEvent, c RawArguments) {
	t.Return <- c
}
func (t *testc) NoArgs(*gateway.MessageCreateEvent) error {
	return errors.New("passed")
}
func (t *testc) OnTyping(*gateway.TypingStartEvent) {
	t.Typed--
}

func TestNewContext(t *testing.T) {
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	c, err := New(s, &testc{})
	if err != nil {
		t.Fatal("Failed to create new context:", err)
	}

	if !reflect.DeepEqual(c.Subcommands(), c.subcommands) {
		t.Fatal("Subcommands mismatch.")
	}
}

func TestContext(t *testing.T) {
	var given = &testc{}
	var s = &state.State{
		Cabinet: store.NoopCabinet,
		Handler: handler.New(),
	}

	sub, err := NewSubcommand(given)
	if err != nil {
		t.Fatal("Failed to create subcommand:", err)
	}

	var ctx = &Context{
		Name:        "arikawa/bot test",
		Description: "Just a test.",

		Subcommand: sub,
		State:      s,
		ParseArgs:  DefaultArgsParser(),
	}

	t.Run("init commands", func(t *testing.T) {
		if err := ctx.Subcommand.InitCommands(ctx); err != nil {
			t.Fatal("Failed to init commands:", err)
		}

		if given.Ctx != ctx {
			t.Fatal("given Context field has invalid pointer")
		}
	})

	t.Run("find commands", func(t *testing.T) {
		cmd := ctx.FindCommand("", "NoArgs")
		if cmd == nil {
			t.Fatal("Failed to find NoArgs")
		}
	})

	t.Run("middleware", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("pls do ")

		// This should trigger the middleware first.
		if err := expect(ctx, given, "3", "pls do getCounter"); err != nil {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("derive intents", func(t *testing.T) {
		intents := ctx.DeriveIntents()

		assertIntents := func(target gateway.Intents, name string) {
			if !intents.Has(target) {
				t.Error("Derived intents do not have", name)
			}
		}

		assertIntents(gateway.IntentGuildMessages, "guild messages")
		assertIntents(gateway.IntentDirectMessages, "direct messages")
		assertIntents(gateway.IntentGuildMessageTyping, "guild typing")
		assertIntents(gateway.IntentDirectMessageTyping, "direct message typing")
	})

	t.Run("typing event", func(t *testing.T) {
		typing := &gateway.TypingStartEvent{}

		if err := ctx.callCmd(typing); err != nil {
			t.Fatal("Failed to call with TypingStart:", err)
		}

		// -1 none ran
		if given.Typed != 1 {
			t.Fatal("Typed bool is false")
		}
	})

	t.Run("call command", func(t *testing.T) {
		// Set a custom prefix
		ctx.HasPrefix = NewPrefix("~")

		var (
			send    = "hacka doll no. 3"
			expects = []string{"hacka", "doll", "no.", "3"}
		)

		if err := expect(ctx, given, expects, "~send "+send); err.Error() != "oh no" {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("call command rawarguments", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("!")
		expects := RawArguments("just things")

		if err := expect(ctx, given, expects, "!content just things"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}
	})

	t.Run("call command custom manual parser", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("!")
		expects := []string{"arg1", ":)"}

		if err := expect(ctx, given, expects, "!custom arg1 :)"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}
	})

	t.Run("call command custom variadic parser", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("!")
		expects := &customParsed{true}

		if err := expect(ctx, given, expects, "!variadic bruh moment"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}
	})

	t.Run("call command custom trailing manual parser", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("!")
		expects := ArgumentParts{"arikawa"}

		if err := sendMsg(ctx, given, &expects, "!trailCustom hime arikawa"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}

		if expects.Length() != 1 {
			t.Fatal("Unexpected ArgumentParts length.")
		}
		if expects.After(1)+expects.After(2)+expects.After(-1) != "" {
			t.Fatal("Unexpected ArgumentsParts after.")
		}
		if expects.String() != "arikawa" {
			t.Fatal("Unexpected ArgumentsParts string.")
		}
		if expects.Arg(0) != "arikawa" {
			t.Fatal("Unexpected ArgumentParts arg 0")
		}
		if expects.Arg(1) != "" {
			t.Fatal("Unexpected ArgumentParts arg 1")
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

		if err == nil || !strings.HasPrefix(err.Error(), "unknown command:") {
			t.Fatal("unexpected error:", err)
		}
	})

	// Test subcommands

	t.Run("register subcommand", func(t *testing.T) {
		ctx.HasPrefix = NewPrefix("run ")

		sub := &testc{}
		ctx.MustRegisterSubcommand(sub)

		if err := testMessage("run testc noop"); err != nil {
			t.Fatal("Unexpected error:", err)
		}

		expects := RawArguments("hackadoll no. 3")

		if err := expect(ctx, sub, expects, "run testc content hackadoll no. 3"); err != nil {
			t.Fatal("Unexpected call error:", err)
		}

		if cmd := ctx.FindCommand("testc", "Noop"); cmd == nil {
			t.Fatal("Failed to find subcommand Noop")
		}
	})

	t.Run("register subcommand custom", func(t *testing.T) {
		ctx.MustRegisterSubcommand(&testc{}, "arikawa", "a")
	})

	t.Run("duplicate subcommand", func(t *testing.T) {
		_, err := ctx.RegisterSubcommand(&testc{}, "arikawa")
		if err := err.Error(); !strings.Contains(err, "duplicate") {
			t.Fatal("Unexpected error:", err)
		}

		_, err = ctx.RegisterSubcommand(&testc{}, "a")
		if err := err.Error(); !strings.Contains(err, "duplicate") {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("help", func(t *testing.T) {
		ctx.MustRegisterSubcommand(&testc{}, "helper")

		h := ctx.Help()
		if h == "" {
			t.Fatal("Empty help?")
		}

		if strings.Contains(h, "hidden") {
			t.Fatal("Hidden command shown in help.")
		}

		if !strings.Contains(h, "arikawa/bot test") {
			t.Fatal("Name not found.")
		}
		if !strings.Contains(h, "Just a test.") {
			t.Fatal("Description not found.")
		}
		if !strings.Contains(h, "**a**") {
			t.Fatal("arikawa alias `a' not found.")
		}
	})

	t.Run("start", func(t *testing.T) {
		cancel := ctx.Start()
		defer cancel()

		ctx.HasPrefix = NewPrefix("!")
		given.Return = make(chan interface{})

		ctx.Handler.Call(&gateway.MessageCreateEvent{
			Message: discord.Message{
				Content: "!content hime arikawa best trap",
			},
		})

		if c := (<-given.Return).(RawArguments); c != "hime arikawa best trap" {
			t.Fatal("Unexpected content:", c)
		}
	})
}

func expect(ctx *Context, given *testc, expects interface{}, content string) (call error) {
	var v interface{}
	if call = sendMsg(ctx, given, &v, content); call != nil {
		return
	}
	if !reflect.DeepEqual(v, expects) {
		return fmt.Errorf("returned argument is invalid: %v", v)
	}
	return nil
}

func sendMsg(ctx *Context, given *testc, into interface{}, content string) (call error) {
	// Return channel for testing
	ret := make(chan interface{})
	given.Return = ret

	// Mock a messageCreate event
	m := &gateway.MessageCreateEvent{
		Message: discord.Message{
			Content: content,
		},
	}

	var callCh = make(chan error)
	go func() {
		callCh <- ctx.Call(m)
	}()

	select {
	case arg := <-ret:
		call = <-callCh
		reflect.ValueOf(into).Elem().Set(reflect.ValueOf(arg))
		return

	case call = <-callCh:
		return fmt.Errorf("expected return before error: %w", call)

	case <-time.After(time.Second):
		return errors.New("timed out while waiting")
	}
}

func BenchmarkConstructor(b *testing.B) {
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	for i := 0; i < b.N; i++ {
		_, _ = New(s, &testc{})
	}
}

func BenchmarkCall(b *testing.B) {
	var given = &testc{}
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	sub, _ := NewSubcommand(given)

	var ctx = &Context{
		Subcommand: sub,
		State:      s,
		HasPrefix:  NewPrefix("~"),
		ParseArgs:  DefaultArgsParser(),
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
	var given = &testc{}
	var s = &state.State{
		Cabinet: store.NoopCabinet,
	}

	sub, _ := NewSubcommand(given)

	var ctx = &Context{
		Subcommand: sub,
		State:      s,
		HasPrefix:  NewPrefix("~"),
		ParseArgs:  DefaultArgsParser(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.Help()
	}
}
