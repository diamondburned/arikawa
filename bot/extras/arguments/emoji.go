package arguments

import (
	"errors"
	"regexp"

	"github.com/diamondburned/arikawa/api/rate"
	"github.com/diamondburned/arikawa/discord"
)

var (
	EmojiRegex = regexp.MustCompile(`<(a?):(.+?):(\d+)>`)

	ErrInvalidEmoji = errors.New("invalid emoji")
)

type Emoji struct {
	ID   discord.Snowflake
	Name string

	Custom   bool
	Animated bool
}

func (e Emoji) APIString() string {
	if !e.Custom {
		return e.Name
	}

	return e.Name + ":" + e.ID.String()
}

func (e Emoji) String() string {
	if !e.Custom {
		return e.Name
	}

	if e.Animated {
		return "<a:" + e.Name + ":" + e.ID.String() + ">"
	} else {
		return "<:" + e.Name + ":" + e.ID.String() + ">"
	}
}

func (e Emoji) URL() string {
	if !e.Custom {
		return ""
	}

	base := "https://cdn.discordapp.com/emojis/" + e.ID.String()

	if e.Animated {
		return base + ".gif"
	} else {
		return base + ".png"
	}
}

func (e *Emoji) Usage() string {
	return "emoji"
}

func (e *Emoji) Parse(arg string) error {
	// Check if Unicode emoji
	if rate.StringIsEmojiOnly(arg) {
		e.Name = arg
		e.Custom = false

		return nil
	}

	var matches = EmojiRegex.FindStringSubmatch(arg)

	if len(matches) != 4 {
		return ErrInvalidEmoji
	}

	id, err := discord.ParseSnowflake(matches[3])
	if err != nil {
		return err
	}

	e.Custom = true
	e.Animated = matches[1] == "a"
	e.Name = matches[2]
	e.ID = id

	return nil
}
