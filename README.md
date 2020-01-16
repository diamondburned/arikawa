# arikawa

![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue?style=flat-square)
 ![Discord nixhub](https://img.shields.io/badge/Discord-nixhub-7289da?style=flat-square)
   ![Hime Arikawa](https://img.shields.io/badge/Hime-Arikawa-ea75a2?style=flat-square)

A Golang library for the Discord API.

## Testing

The package includes integration tests that require `$BOT_TOKEN`. To run these
tests, do

```sh
export BOT_TOKEN="<BOT_TOKEN>"
go test -tags integration ./...
```
