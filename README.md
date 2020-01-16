# arikawa

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue?style=flat-square           )](https://godoc.org/github.com/diamondburned/arikawa)
[![       Examples](https://img.shields.io/badge/Example-__example%2F-blueviolet?style=flat-square)]()
[![ Discord nixhub](https://img.shields.io/badge/Discord-nixhub-7289da?style=flat-square          )](https://discord.gg/kF9mYBV                        )
[![   Hime Arikawa](https://img.shields.io/badge/Hime-Arikawa-ea75a2?style=flat-square            )](https://hime-goto.fandom.com/wiki/Hime_Arikawa    )

A Golang library for the Discord API.

## Testing

The package includes integration tests that require `$BOT_TOKEN`. To run these
tests, do

```sh
export BOT_TOKEN="<BOT_TOKEN>"
go test -tags integration ./...
```
