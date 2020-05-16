package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/diamondburned/arikawa/discord"
)

func TestMarshalAllowedMentions(t *testing.T) {
	t.Run("parse nothing", func(t *testing.T) {
		var data = SendMessageData{
			AllowedMentions: &AllowedMentions{
				Parse: []AllowedMentionType{},
			},
		}

		if j := mustMarshal(t, data); j != `{"allowed_mentions":{"parse":[]}}` {
			t.Fatal("Unexpected JSON:", j)
		}
	})

	t.Run("allow everything", func(t *testing.T) {
		var data = SendMessageData{
			Content: "a",
		}

		if j := mustMarshal(t, data); j != `{"content":"a"}` {
			t.Fatal("Unexpected JSON:", j)
		}
	})

	t.Run("allow certain user IDs", func(t *testing.T) {
		var data = SendMessageData{
			AllowedMentions: &AllowedMentions{
				Users: []discord.Snowflake{1, 2},
			},
		}

		if j := mustMarshal(t, data); j != `{"allowed_mentions":{"parse":null,"users":["1","2"]}}` {
			t.Fatal("Unexpected JSON:", j)
		}
	})
}

func TestVerifyAllowedMentions(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var am = AllowedMentions{
			Parse: []AllowedMentionType{AllowEveryoneMention, AllowUserMention},
			Users: []discord.Snowflake{69, 420},
		}

		err := am.Verify()
		errMustContain(t, err, "Users slice is not empty")
	})

	t.Run("users too long", func(t *testing.T) {
		var am = AllowedMentions{
			Users: make([]discord.Snowflake, 101),
		}

		err := am.Verify()
		errMustContain(t, err, "users slice length 101 is over 100")
	})

	t.Run("roles too long", func(t *testing.T) {
		var am = AllowedMentions{
			Roles: make([]discord.Snowflake, 101),
		}

		err := am.Verify()
		errMustContain(t, err, "roles slice length 101 is over 100")
	})

	t.Run("valid", func(t *testing.T) {
		var am = AllowedMentions{
			Parse: []AllowedMentionType{AllowEveryoneMention, AllowUserMention},
			Roles: []discord.Snowflake{1337},
			Users: []discord.Snowflake{},
		}

		if err := am.Verify(); err != nil {
			t.Fatal("Unexpected error:", err)
		}
	})
}

func TestSendMessage(t *testing.T) {
	send := func(data SendMessageData) error {
		// A nil client will cause a panic.
		defer func() {
			recover()
		}()

		// shouldn't matter
		client := (*Client)(nil)
		_, err := client.SendMessageComplex(0, data)
		return err
	}

	t.Run("empty", func(t *testing.T) {
		var empty = SendMessageData{
			Content: "",
			Embed:   nil,
		}

		if err := send(empty); err != ErrEmptyMessage {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("files only", func(t *testing.T) {
		var empty = SendMessageData{
			Files: []SendMessageFile{{Name: "test.jpg"}},
		}

		if err := send(empty); err != nil {
			t.Fatal("Unexpected error:", err)
		}
	})

	t.Run("invalid allowed mentions", func(t *testing.T) {
		var data = SendMessageData{
			Content: "hime arikawa",
			AllowedMentions: &AllowedMentions{
				Parse: []AllowedMentionType{AllowEveryoneMention, AllowUserMention},
				Users: []discord.Snowflake{69, 420},
			},
		}

		err := send(data)
		errMustContain(t, err, "allowedMentions error")
	})

	t.Run("invalid embed", func(t *testing.T) {
		var data = SendMessageData{
			Embed: &discord.Embed{
				// max 256
				Title: spaces(257),
			},
		}

		err := send(data)
		errMustContain(t, err, "embed error")
	})
}

func errMustContain(t *testing.T, err error, contains string) {
	// mark function as helper so line traces are accurate.
	t.Helper()

	if err != nil && strings.Contains(err.Error(), contains) {
		return
	}
	t.Fatal("Unexpected error:", err)
}

func spaces(length int) string {
	return strings.Repeat(" ", length)
}

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()

	j, err := json.Marshal(v)
	if err != nil {
		t.Fatal("Failed to marshal data:", err)
	}
	return string(j)
}
