# arikawa

[![ Pipeline Status ][pipeline_img    ]][pipeline    ]
[![ Report Card     ][goreportcard_img]][goreportcard]
[![ Godoc Reference ][pkg.go.dev_img  ]][pkg.go.dev  ]
[![ Examples        ][examples_img    ]][examples    ]
[![ Discord Gophers ][dgophers_img    ]][dgophers    ]
[![ Hime Arikawa    ][himeArikawa_img ]][himeArikawa ]

A Golang library for the Discord API.

[dgophers]:     https://discord.gg/7jSf85J
[dgophers_img]: https://img.shields.io/badge/Discord%20Gophers-%23arikawa-%237289da?style=flat-square

[examples]:     https://github.com/diamondburned/arikawa/tree/v3/0-examples
[examples_img]: https://img.shields.io/badge/Example-.%2F0--examples%2F-blueviolet?style=flat-square

[pipeline]:     https://github.com/diamondburned/arikawa/actions/workflows/test.yml
[pipeline_img]: https://img.shields.io/github/actions/workflow/status/diamondburned/arikawa/test.yml?style=flat-square&label=Tests

[pkg.go.dev]:     https://pkg.go.dev/github.com/diamondburned/arikawa/v3
[pkg.go.dev_img]: https://img.shields.io/badge/%E2%80%8B-reference-007d9c?logo=go&logoColor=white&style=flat-square

[himeArikawa]:     https://hime-goto.fandom.com/wiki/Hime_Arikawa
[himeArikawa_img]: https://img.shields.io/badge/Hime-Arikawa-ea75a2?style=flat-square

[goreportcard]:     https://goreportcard.com/report/github.com/diamondburned/arikawa
[goreportcard_img]: https://goreportcard.com/badge/github.com/diamondburned/arikawa?style=flat-square&label=Go%20Report


## Library Highlights

- More modularity with components divided up into independent packages, such as
  the API client and the Websocket Gateway being fully independent.
- Clear separation of models: API and Gateway models are never mixed together so
  to not be confusing.
- Extend and intercept Gateway events, allowing for use cases such as reading
  deleted messages.
- Pluggable Gateway cache allows for custom caching implementations such as
  Redis, automatically falling back to the API if needed.
- Typed Snowflakes make it much harder to accidentally use the wrong ID (e.g.
  it is impossible to use a channel ID as a message ID).
- Working user account support, with much of them in [ningen][ningen]. Please
  do not use this for self-botting, as that is against Discord's ToS.

[ningen]: https://github.com/diamondburned/ningen


## Examples

### [Commands (Hybrid)](https://github.com/diamondburned/arikawa/tree/v3/0-examples/commands-hybrid)

commands-hybrid is an alternative variant of
[commands](https://github.com/diamondburned/arikawa/tree/v3/0-examples/commands),
where the program permits being hosted either as a Gateway-based daemon or as a
web server using the Interactions Webhook API.

Both examples demonstrate adding interaction commands into the bot as well as an
example of routing those commands to be executed.

### [Simple](https://github.com/diamondburned/arikawa/tree/v3/0-examples/simple)

Simple bot example without any state. All it does is logging messages sent into
the console. Run with `BOT_TOKEN="TOKEN" go run .`. This example only
demonstrates the most simple needs; in most cases, bots should use the state or
the bot router.

**Note** that Discord discourages use of bots that do not use the interactions
API, meaning that this example should not be used for bots.

### [Undeleter](https://github.com/diamondburned/arikawa/tree/v3/0-examples/undeleter)

A slightly more complicated example. This bot uses a local state to cache
everything, including messages. It detects when someone deletes a message,
logging the content into the console.

This example demonstrates the PreHandler feature of the state library.
PreHandler calls all handlers that are registered (separately from the session),
calling them before the state is updated.

**Note** that Discord discourages use of bots that do not use the interactions
API, meaning that this example should not be used for bots.

### Bare Minimum Bot Example

The least amount of code recommended to have a bot that responds to a /ping.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var commands = []api.CreateCommandData{{Name: "ping", Description: "Ping!"}}

func main() {
	r := cmdroute.NewRouter()
	r.AddFunc("ping", func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
		return &api.InteractionResponseData{Content: option.NewNullableString("Pong!")}
	})

	s := state.New("Bot " + os.Getenv("BOT_TOKEN"))
	s.AddInteractionHandler(r)
	s.AddIntents(gateway.IntentGuilds)

	if err := cmdroute.OverwriteCommands(s, commands); err != nil {
		log.Fatalln("cannot update commands:", err)
	}

	if err := s.Connect(context.TODO()); err != nil {
		log.Println("cannot connect:", err)
	}
}
```


## Testing

The package includes integration tests that require `$BOT_TOKEN`. To run these
tests, do:

```sh
export BOT_TOKEN="<BOT_TOKEN>"
go test -tags integration -race ./...
```
