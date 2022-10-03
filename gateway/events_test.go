package gateway

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

func TestRequestGuildMembersCommand(t *testing.T) {
	assert := func(cmd Event, data map[string]interface{}) {
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatal("failed to marshal command:", err)
		}

		var cmdMap map[string]interface{}
		if err := json.Unmarshal(cmdBytes, &cmdMap); err != nil {
			t.Fatal("failed to unmarshal command:", err)
		}

		if !reflect.DeepEqual(cmdMap, data) {
			t.Fatalf("mismatched command, got %#v", cmdMap)
		}
	}

	t.Run("userIDs", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			UserIDs:  []discord.UserID{456},
		}

		assert(&cmd, map[string]interface{}{
			"guild_ids": []interface{}{"123"},
			"user_ids":  []interface{}{"456"},
			"presences": false,
		})
	})

	t.Run("query_empty", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			Query:    option.NewString(""),
		}

		assert(&cmd, map[string]interface{}{
			"guild_ids": []interface{}{"123"},
			"query":     "",
			"limit":     float64(0),
			"presences": false,
		})
	})

	t.Run("query_nonempty", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			Query:    option.NewString("abc"),
		}

		assert(&cmd, map[string]interface{}{
			"guild_ids": []interface{}{"123"},
			"query":     "abc",
			"limit":     float64(0),
			"presences": false,
		})
	})

	t.Run("both", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			UserIDs:  []discord.UserID{456},
			Query:    option.NewString("abc"),
		}

		// Gateway should never be touched when Marshal fails, so we can just
		// create a zero-value.
		var gateway ws.Gateway

		err := gateway.Send(context.Background(), &cmd)
		if err == nil || !strings.Contains(err.Error(), "neither UserIDs nor Query can be filled") {
			t.Fatal("unexpected error:", err)
		}
	})
}
