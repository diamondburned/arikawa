package arguments

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

func TestEmojiRune(t *testing.T) {
	const emoji = "💩"

	e := Emoji{}
	if err := e.Parse(emoji); err != nil {
		t.Fatal("Failed to parse emoji:", err)
	}

	if u := e.URL(); u != "" {
		t.Fatal("Unexpected URL:", u)
	}

	if s := e.String(); s != emoji {
		t.Fatal("Unexpected string:", s)
	}

	if s := e.APIString(); s != emoji {
		t.Fatal("Unexpected API string:", s)
	}
}

func TestEmojiCustom(t *testing.T) {
	const emoji = "<:StareNeutral:612368399732965376>"
	const url = "https://cdn.discordapp.com/emojis/612368399732965376.png"

	e := Emoji{}
	if err := e.Parse(emoji); err != nil {
		t.Fatal("Failed to parse emoji:", err)
	}

	if u := e.URL(); u != url {
		t.Fatal("Unexpected URL:", u)
	}

	if s := e.String(); s != emoji {
		t.Fatal("Unexpected string:", s)
	}

	if s := e.APIString(); s != "StareNeutral:612368399732965376" {
		t.Fatal("Unexpected API string:", s)
	}
}

func TestEmojiAnimated(t *testing.T) {
	const emoji = "<a:StareNodGIF:614322540332056577>"
	const url = "https://cdn.discordapp.com/emojis/614322540332056577.gif"

	e := Emoji{}
	if err := e.Parse(emoji); err != nil {
		t.Fatal("Failed to parse emoji:", err)
	}

	if u := e.URL(); u != url {
		t.Fatal("Unexpected URL:", u)
	}

	if s := e.String(); s != emoji {
		t.Fatal("Unexpected string:", s)
	}

	if s := e.APIString(); s != "StareNodGIF:614322540332056577" {
		t.Fatal("Unexpected API string:", s)
	}
}
