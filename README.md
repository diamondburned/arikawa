# arikawa

[![ Pipeline Status ][pipeline_img    ]][pipeline    ]
[![ Coverage        ][coverage_img    ]][pipeline    ]
[![ Report Card     ][goreportcard_img]][goreportcard]
[![ Godoc Reference ][pkg.go.dev_img  ]][pkg.go.dev  ]
[![ Examples        ][examples_img    ]][examples    ]
[![ Discord Gophers ][dgophers_img    ]][dgophers    ]
[![ Hime Arikawa    ][himeArikawa_img ]][himeArikawa ]

A Golang library for the Discord API.

[dgophers]:     https://discord.gg/7jSf85J
[dgophers_img]: https://img.shields.io/badge/Discord%20Gophers-%23arikawa-%237289da?style=flat-square

[examples]:     https://github.com/diamondburned/arikawa/tree/v3/_example
[examples_img]: https://img.shields.io/badge/Example-__example%2F-blueviolet?style=flat-square

[pipeline]:     https://gitlab.com/diamondburned/arikawa/pipelines
[pipeline_img]: https://gitlab.com/diamondburned/arikawa/badges/v3/pipeline.svg?style=flat-square
[coverage_img]: https://gitlab.com/diamondburned/arikawa/badges/v3/coverage.svg?style=flat-square

[pkg.go.dev]:     https://pkg.go.dev/github.com/diamondburned/arikawa/v3
[pkg.go.dev_img]: https://pkg.go.dev/badge/github.com/diamondburned/arikawa/v3

[himeArikawa]:     https://hime-goto.fandom.com/wiki/Hime_Arikawa
[himeArikawa_img]: https://img.shields.io/badge/Hime-Arikawa-ea75a2?style=flat-square

[goreportcard]:     https://goreportcard.com/report/github.com/diamondburned/arikawa
[goreportcard_img]: https://goreportcard.com/badge/github.com/diamondburned/arikawa?style=flat-square


## Examples

### [Simple](https://github.com/diamondburned/arikawa/tree/v3/_example/simple)

Simple bot example without any state. All it does is logging messages sent into
the console. Run with `BOT_TOKEN="TOKEN" go run .`. This example only
demonstrates the most simple needs; in most cases, bots should use the state or
the bot router.

### [Undeleter](https://github.com/diamondburned/arikawa/tree/v3/_example/undeleter)

A slightly more complicated example. This bot uses a local state to cache
everything, including messages. It detects when someone deletes a message,
logging the content into the console.

This example demonstrates the PreHandler feature of the state library.
PreHandler calls all handlers that are registered (separately from the session),
calling them before the state is updated.

### Bare Minimum Bot

The least amount of code for a basic ping-pong bot. It's similar to Serenity's
Discord bot example in the README.

```go
package main

import (
	"os"

	"github.com/diamondburned/arikawa/v3/bot"
	"github.com/diamondburned/arikawa/v3/gateway"
)

func main() {
	bot.Run(os.Getenv("DISCORD_TOKEN"), &Bot{},
		func(ctx *bot.Context) error {
			ctx.HasPrefix = bot.NewPrefix("!")
			return nil
		},
	)
}

type Bot struct {
	Ctx *bot.Context
}

func (b *Bot) Ping(*gateway.MessageCreateEvent) (string, error) {
	return "Pong!", nil
}
```


## Where is package `bot`?

Package bot has now been deprecated after Discord's decision to eventually
deprecate regular message events as means of commanding bots. We've decided to
move the old `bot` package into `utils/` to signify that it should no longer be
used.

Moving `bot` into `utils/` will allow us to eventually rewrite the whole package
to use slash commands without worrying about breaking the old (v2) API, which is
great, because almost nothing translates well from the previous design to slash
commands.


## Comparison: Why not discordgo?

Discordgo is great. It's the first library that I used when I was learning Go.
Though there are some things that I disagree on. Here are some ways that this
library is different:

- Better package structure: this library divides the Discord library up into
smaller packages.
- Cleaner API/Gateway structure separation: this library separates fields that
would only appear in Gateway events, so to not cause confusion.
- Automatic un-pagination: this library automatically un-paginates endpoints
that would otherwise not return everything fully.
- Flexible underlying abstractions: this library allows plugging in different
JSON and Websocket implementations, as well as direct access to the HTTP 
client.
- Flexible API abstractions: because packages are separated, the developer could
choose to use a lower level package (such as `gateway`) or a higher level
package (such as `state`).
- Pre-handlers in the state: this allows the developers to access items from the
state storage before they're removed.
- Pluggable state storages: although only having a default state storage in the
library, it is abstracted with an interface, making it possible to implement a
custom remote or local state storage.
- REST-updated state: this library will call the REST API if it can't find
things in the state, which is useful for keeping it updated.
- No code generation: just so the library is a lot easier to maintain.


## Testing

The package includes integration tests that require `$BOT_TOKEN`. To run these
tests, do:

```sh
export BOT_TOKEN="<BOT_TOKEN>"
go test -tags integration -race ./...
```
