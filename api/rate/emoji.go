package rate

import (
	"strconv"
	"strings"
	"unicode/utf16"
)

func StringIsEmojiOnly(emoji string) bool {
	runes := []rune(emoji)
	// Slice of runes is 2, since some emojis have 2 runes.
	switch len(runes) {
	case 0:
		return false
	case 1, 2:
		return EmojiRune(runes[0])
	// case 2:
	// return EmojiRune(runes[0]) && EmojiRune(runes[1])
	default:
		return false
	}
}

func StringIsCustomEmoji(emoji string) bool {
	parts := strings.Split(emoji, ":")
	if len(parts) != 2 {
		return false
	}

	// Validate ID
	_, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// Validate name, shouldn't have whitespaces
	if strings.ContainsRune(parts[0], ' ') {
		return false
	}

	return true
}

var surrogates = [...][2]rune{ // [0] from, [1] to
	{utf16.DecodeRune(0xD83C, 0xD000), utf16.DecodeRune(0xD83C, 0xDFFF)},
	{utf16.DecodeRune(0xD83E, 0xD000), utf16.DecodeRune(0xD83E, 0xDFFF)},
	{utf16.DecodeRune(0xD83F, 0xD000), utf16.DecodeRune(0xD83F, 0xDFFF)},
}

func EmojiRune(r rune) bool {
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
