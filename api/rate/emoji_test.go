// +build unit

package rate

import "testing"

func TestEmojiRuneParsing(t *testing.T) {
	var emojis = []string{
		"🏑",
		"❄️",
		"🤲🏿",
	}

	var notEmojis = []string{
		"🏃🏿🏃🏿", // dual emojis
		"te",   // not emoji
	}

	for i, emoji := range emojis {
		if !StringIsEmojiOnly(emoji) {
			t.Fatal(i, "is an emoji, function returned false")
		}
	}

	for i, not := range notEmojis {
		if StringIsEmojiOnly(not) {
			t.Fatal(i, "is not an emoji, function returned true")
		}
	}
}

func TestEmojiCustomParsing(t *testing.T) {
	var emojis = []string{
		"emoji_thing:213131141",
		"StareNeutral:612368399732965376",
	}

	for i, emoji := range emojis {
		if !StringIsCustomEmoji(emoji) {
			t.Fatal(i, "is a custom emoji, fn returned false")
		}
	}
}
