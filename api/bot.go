package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

// BotData contains the GatewayURL as well as extra metadata on how to
// shard bots.
type BotData struct {
	URL        string             `json:"url"`
	Shards     int                `json:"shards,omitempty"`
	StartLimit *SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit is the information on the current session start limit. It's
// used in BotData.
type SessionStartLimit struct {
	Total          int                  `json:"total"`
	Remaining      int                  `json:"remaining"`
	ResetAfter     discord.Milliseconds `json:"reset_after"`
	MaxConcurrency int                  `json:"max_concurrency"`
}

// BotURL fetches the Gateway URL along with extra metadata. The token
// passed in will NOT be prefixed with Bot.
func (c *Client) BotURL() (*BotData, error) {
	var g *BotData
	return g, c.RequestJSON(&g, "GET", EndpointGatewayBot)
}

// GatewayURL asks Discord for a Websocket URL to the Gateway.
func GatewayURL() (string, error) {
	var g BotData
	return g.URL, httputil.NewClient().RequestJSON(&g, "GET", EndpointGateway)
}
