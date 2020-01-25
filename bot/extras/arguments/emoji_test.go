package arguments

import "testing"

func TestEmojiRune(t *testing.T) {
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
		if !stringIsEmojiOnly(emoji) {
			t.Fatal(i, "is an emoji, function returned false")
		}
	}

	for i, not := range notEmojis {
		if stringIsEmojiOnly(not) {
			t.Fatal(i, "is not an emoji, function returned true")
		}
	}
}
