package arguments

import (
	"errors"
	"regexp"
	"unicode/utf16"
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

func (e *Emoji) Parse(arg string) error {
	// Check if Unicode emoji
	if stringIsEmojiOnly(arg) {
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

func stringIsEmojiOnly(emoji string) bool {
	runes := []rune(emoji)
	// Slice of runes is 2, since some emojis have 2 runes.
	if len(runes) > 2 {
		return false
	}

	return emojiRune(runes[0])
}

var surrogates = [...][2]rune{ // [0] from, [1] to
	{utf16.DecodeRune(0xD83C, 0xD000), utf16.DecodeRune(0xD83C, 0xDFFF)},
	{utf16.DecodeRune(0xD83E, 0xD000), utf16.DecodeRune(0xD83E, 0xDFFF)},
	{utf16.DecodeRune(0xD83F, 0xD000), utf16.DecodeRune(0xD83F, 0xDFFF)},
}

func emojiRune(r rune) bool {
	b := r == '\u00a9' || r == '\u00ae' ||
		(r >= '\u2000' && r <= '\u3300')
	if b {
		return true
	}

	for _, surrogate := range surrogates {
		if surrogate[0] <= r && r <= surrogate[1] {
			return true
		}
	}

	return false
}
