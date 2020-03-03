package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/bot"
	"github.com/diamondburned/arikawa/bot/extras/arguments"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
)

type Bot struct {
	// Context must not be embedded.
	Ctx *bot.Context
}

// Help prints the default help message.
func (bot *Bot) Help(m *gateway.MessageCreateEvent) (string, error) {
	return bot.Ctx.Help(), nil
}

// Add demonstrates the usage of typed arguments. Run it with "~add 1 2".
func (bot *Bot) Add(m *gateway.MessageCreateEvent, a, b int) error {
	content := fmt.Sprintf("%d + %d = %d", a, b, a+b)

	_, err := bot.Ctx.SendMessage(m.ChannelID, content, nil)
	return err
}

// Ping is a simple ping example, perhaps the most simple you could make it.
func (bot *Bot) Ping(m *gateway.MessageCreateEvent) error {
	_, err := bot.Ctx.SendMessage(m.ChannelID, "Pong!", nil)
	return err
}

// Say demonstrates how arguments.Flag could be used without the flag library.
func (bot *Bot) Say(
	m *gateway.MessageCreateEvent, f *arguments.Flag) (string, error) {

	args := f.String()
	if args == "" {
		// Empty message, ignore
		return "", nil
	}

	return args, nil
}

// GuildInfo demonstrates the use of command flags, in this case the GuildOnly
// flag.
func (bot *Bot) G_GuildInfo(m *gateway.MessageCreateEvent) (string, error) {
	g, err := bot.Ctx.Guild(m.GuildID)
	if err != nil {
		return "", fmt.Errorf("Failed to get guild: %v", err)
	}

	return fmt.Sprintf(
		"Your guild is %s, and its maximum members is %d",
		g.Name, g.MaxMembers,
	), nil
}

// Repeat tells the bot to wait for the user's response, then repeat what they
// said.
func (bot *Bot) Repeat(m *gateway.MessageCreateEvent) (string, error) {
	_, err := bot.Ctx.SendMessage(m.ChannelID,
		"What do you want me to say?", nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	v := bot.Ctx.WaitFor(ctx, func(v interface{}) bool {
		// Incoming event is a message create event:
		mg, ok := v.(*gateway.MessageCreateEvent)
		if !ok {
			return false
		}

		// Message is from the same author:
		return mg.Author.ID == m.Author.ID
	})

	if v == nil {
		return "", errors.New("Timed out waiting for response.")
	}

	ev := v.(*gateway.MessageCreateEvent)
	return ev.Content, nil
}

// Embed is a simple embed creator. Its purpose is to demonstrate the usage of
// the ParseContent interface, as well as using the stdlib flag package.
func (bot *Bot) Embed(
	m *gateway.MessageCreateEvent, f *arguments.Flag) (*discord.Embed, error) {

	fs := arguments.NewFlagSet()

	var (
		title  = fs.String("title", "", "Title")
		author = fs.String("author", "", "Author")
		footer = fs.String("footer", "", "Footer")
		color  = fs.String("color", "#FFFFFF", "Color in hex format #hhhhhh")
	)

	if err := f.With(fs.FlagSet); err != nil {
		return nil, err
	}

	if len(fs.Args()) < 1 {
		return nil, fmt.Errorf("Usage: embed [flags] content...\n" + fs.Usage())
	}

	// Check if the color string is valid.
	if !strings.HasPrefix(*color, "#") || len(*color) != 7 {
		return nil, errors.New("Invalid color, format must be #hhhhhh")
	}

	// Parse the color into decimal numbers.
	colorHex, err := strconv.ParseInt((*color)[1:], 16, 64)
	if err != nil {
		return nil, err
	}

	// Make a new embed
	embed := discord.Embed{
		Title:       *title,
		Description: strings.Join(fs.Args(), " "),
		Color:       discord.Color(colorHex),
	}

	if *author != "" {
		embed.Author = &discord.EmbedAuthor{
			Name: *author,
		}
	}
	if *footer != "" {
		embed.Footer = &discord.EmbedFooter{
			Text: *footer,
		}
	}

	return &embed, err
}
