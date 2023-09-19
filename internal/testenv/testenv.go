//go:build !uintonly
// +build !uintonly

package testenv

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

const PerseveranceTime = 50 * time.Minute

type Env struct {
	BotToken   string
	ChannelID  discord.ChannelID
	VoiceChID  discord.ChannelID
	ShardCount int // default 3
}

var (
	globalEnv Env
	globalErr error
	once      sync.Once
)

func Must(t *testing.T) Env {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	e, err := GetEnv()
	if err != nil {
		t.Skip("integration test variables missing")
	}

	return e
}

func GetEnv() (Env, error) {
	once.Do(getEnv)
	return globalEnv, globalErr
}

func getEnv() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		globalErr = errors.New("missing $BOT_TOKEN")
		return
	}

	id, err := discord.ParseSnowflake(os.Getenv("CHANNEL_ID"))
	if err != nil {
		globalErr = fmt.Errorf("invalid $CHANNEL_ID: %w", err)
		return
	}

	vid, err := discord.ParseSnowflake(os.Getenv("VOICE_ID"))
	if err != nil {
		globalErr = fmt.Errorf("invalid $VOICE_ID: %w", err)
		return
	}

	shardCount := 2
	if c, err := strconv.Atoi(os.Getenv("SHARD_COUNT")); err == nil {
		shardCount = c
	}

	globalEnv = Env{
		BotToken:   token,
		ChannelID:  discord.ChannelID(id),
		VoiceChID:  discord.ChannelID(vid),
		ShardCount: shardCount,
	}
}
