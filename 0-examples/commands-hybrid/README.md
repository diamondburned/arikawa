# commands-hybrid

commands-hybrid is an alternative variant of commands, where the program permits
being hosted either as a Gateway-based daemon or as a web server using the
Interactions Webhook API.

## Usage

### Gateway Mode

```sh
BOT_TOKEN="<token here>" go run .
```

### Interactions Webhook Mode

```sh
BOT_TOKEN="<token here>" WEBHOOK_ADDR="localhost:29485" WEBHOOK_PUBKEY="<hex app pubkey>" go run .
```

The endpoint will be `http://localhost:29485/`. I recommend using something like
[srv.us](https://srv.us) to expose this endpoint as a public one, which can then
be used by Discord.
