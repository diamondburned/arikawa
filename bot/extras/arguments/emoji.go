package arguments

import (
	"errors"
	"regexp"
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

func (e *Emoji) Parse(arg string) error {
	// Check if Unicode
	var unicode string

	for _, r := range arg {
		if r < '\U0001F600' && r > '\U0001F64F' {
			unicode += string(r)
		}
	}

	if unicode != "" {
		e.ID = unicode
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
