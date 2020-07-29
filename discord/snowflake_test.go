package discord

import (
	"testing"
	"time"
)

func TestSnowflake(t *testing.T) {
	t.Run("parse", func(t *testing.T) {
		_, err := ParseSnowflake("175928847299117063")
		if err != nil {
			t.Fatal("Failed to parse snowflake:", err)
		}
	})

	const value = 175928847299117063
	var expect = time.Date(2016, 04, 30, 11, 18, 25, 796*int(time.Millisecond), time.UTC)

	t.Run("methods", func(t *testing.T) {
		s := Snowflake(value)

		if ts := s.Time(); !ts.Equal(expect) {
			t.Fatal("Unexpected time (expected/got):", expect, ts)
		}

		if s.Worker() != 1 {
			t.Fatal("Unexpected worker:", s.Worker())
		}

		if s.PID() != 0 {
			t.Fatal("Unexpected PID:", s.PID())
		}

		if s.Increment() != 7 {
			t.Fatal("Unexpected increment:", s.Increment())
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		t.Run("0", func(t *testing.T) {
			if Snowflake(0).IsValid() {
				t.Fatal("0 isn't a valid Snowflake")
			}
		})

		t.Run("null", func(t *testing.T) {
			if NullSnowflake.IsValid() {
				t.Fatal("NullSnowflake isn't a valid Snowflake")
			}
		})

		t.Run("valid", func(t *testing.T) {
			var testFlake Snowflake = 123

			if !testFlake.IsValid() {
				t.Fatal(testFlake, "is a valid Snowflake")
			}
		})
	})

	t.Run("new", func(t *testing.T) {
		if s := NewSnowflake(expect); !s.Time().Equal(expect) {
			t.Fatal("Unexpected new snowflake from expected time:", s)
		}
	})
}
