package arguments

import (
	"reflect"
	"strings"
	"testing"
)

func TestFlagSet(t *testing.T) {
	fs := NewFlagSet()

	var b bool
	fs.BoolVar(&b, "b", false, "Test boolean")

	if usage := fs.Usage(); !strings.Contains(usage, "Test boolean") {
		t.Fatal("Unexpected usage:", usage)
	}

	if err := fs.Parse([]string{"-b", "asdasd"}); err != nil {
		t.Fatal("Failed to parse:", err)
	}

	if !b {
		t.Fatal("Test boolean is false")
	}
}

func TestFlag(t *testing.T) {
	f := Flag{}

	if err := f.ParseContent([]string{"gc", "--now", "1m4s"}); err != nil {
		t.Fatal("Failed to parse:", err)
	}

	if f.Command() != "gc" {
		t.Fatal("Unexpected command:", f.Command())
	}

	if args := f.Args(); !reflect.DeepEqual(args, []string{"--now", "1m4s"}) {
		t.Fatal("Unexpected arguments:", args)
	}

	if arg := f.Arg(1200); arg != "" {
		t.Fatal("Unexpected argument at 1200th:", arg)
	}

	if arg := f.Arg(0); arg != "--now" {
		t.Fatal("Unexpected argument at 1st:", arg)
	}

	if s := f.String(); s != "--now 1m4s" {
		t.Fatal("Unexpected string:", s)
	}

	fs := NewFlagSet()

	var now bool
	fs.BoolVar(&now, "now", false, "Now")

	if err := f.With(fs.FlagSet); err != nil {
		t.Fatal("Failed to parse:", err)
	}

	if !now {
		t.Fatal("now is false")
	}

	if arg := fs.FlagSet.Arg(0); arg != "1m4s" {
		t.Fatal("Unexpected argument:", arg)
	}
}
