package arguments

import (
	"errors"
	"regexp"

	"github.com/diamondburned/arikawa/api/rate"
)

var (
	EmojiRegex = regexp.MustCompile(`<(a?):(.+?):(\d+)>`)

	ErrInvalidEmoji = errors.New("Invalid emoji")
)

type Emoji struct {
	ID string

	Custom   bool
	Name     string
	Animated bool
}

func (e Emoji) APIString() string {
	if !e.Custom {
		return e.ID
	}

	return e.Name + ":" + e.ID
}

func (e Emoji) String() string {
	if !e.Custom {
		return e.ID
	}

	if e.Animated {
		return "<a:" + e.Name + ":" + e.ID + ">"
	} else {
		return "<:" + e.Name + ":" + e.ID + ">"
	}
}

func (e Emoji) URL() string {
	if !e.Custom {
		return ""
	}

	base := "https://cdn.discordapp.com/emojis/" + e.ID

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
		e.ID = arg
		e.Custom = false

		return nil
	}

	var matches = EmojiRegex.FindStringSubmatch(arg)

	if len(matches) != 4 {
		return ErrInvalidEmoji
	}

	e.Custom = true
	e.Animated = matches[1] == "a"
	e.Name = matches[2]
	e.ID = matches[3]

	return nil
}
