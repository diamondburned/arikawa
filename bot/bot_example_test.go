package bot_test

import (
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/bot"
)

type duration time.Duration

func (d *duration) Parse(s string) error {
	t, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = duration(t)
	return nil
}

type Main struct {
	*bot.State
}

type pingArgs struct {
	Delay duration
}

func (m *Main) Ping(args pingArgs) api.InteractionResponseData {
}

type echoArgs struct {
	Message string
}

func (m *Main) Echo(args echoArgs) (string, error) {}
