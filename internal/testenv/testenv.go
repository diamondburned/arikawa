// +build !uintonly

package testenv

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/pkg/errors"
)

const PerseveranceTime = 50 * time.Minute

type Env struct {
	BotToken  string
	ChannelID discord.ChannelID
	VoiceChID discord.ChannelID
}

var (
	globalEnv Env
	globalErr error
	once      sync.Once
)

func Must(t *testing.T) Env {
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
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		globalErr = errors.New("missing $BOT_TOKEN")
		return
	}

	var cid = os.Getenv("CHANNEL_ID")
	if cid == "" {
		globalErr = errors.New("missing $CHANNEL_ID")
		return
	}

	id, err := discord.ParseSnowflake(cid)
	if err != nil {
		globalErr = errors.Wrap(err, "invalid $CHANNEL_ID")
		return
	}

	var sid = os.Getenv("VOICE_ID")
	if sid == "" {
		globalErr = errors.New("missing $VOICE_ID")
		return
	}

	vid, err := discord.ParseSnowflake(sid)
	if err != nil {
		globalErr = errors.Wrap(err, "invalid $VOICE_ID")
		return
	}

	globalEnv = Env{
		BotToken:  token,
		ChannelID: discord.ChannelID(id),
		VoiceChID: discord.ChannelID(vid),
	}
}
