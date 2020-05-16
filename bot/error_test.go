package bot

import (
	"errors"
	"strings"
	"testing"
)

func TestInvalidUsage(t *testing.T) {
	t.Run("fmt", func(t *testing.T) {
		err := ErrInvalidUsage{
			Prefix: "!",
			Args:   []string{"hime", "arikawa"},
			Index:  1,
			Wrap:   errors.New("test error"),
		}
		str := err.Error()

		if !strings.Contains(str, "test error") {
			t.Fatal("does not contain 'test error':", str)
		}

		if !strings.Contains(str, "__arikawa__") {
			t.Fatal("Unexpected highlight index:", str)
		}
	})

	t.Run("missing arguments", func(t *testing.T) {
		err := ErrInvalidUsage{}
		str := err.Error()

		if str != "missing arguments. Refer to help." {
			t.Fatal("Unexpected error:", str)
		}
	})

	t.Run("no index", func(t *testing.T) {
		err := ErrInvalidUsage{Wrap: errors.New("astolfo")}
		str := err.Error()

		if str != "invalid usage, error: astolfo." {
			t.Fatal("Unexpected error:", str)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		var err = errors.New("hackadoll no. 3")
		var wrap = &ErrInvalidUsage{
			Wrap: err,
		}

		if !errors.Is(wrap, err) {
			t.Fatal("Failed to unwrap, errors mismatch.")
		}
	})
}
