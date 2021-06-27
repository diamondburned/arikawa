package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

type SearchData struct {
	Offset    uint              `schema:"offset,omitempty"`
	Content   string            `schema:"content,omitempty"`
	Has       string            `schema:"has,omitempty"`
	SortBy    string            `schema:"sort_by,omitempty"`
	SortOrder string            `schema:"sort_order,omitempty"`
	ChannelID discord.ChannelID `schema:"channel_id,omitempty"`
	AuthorID  discord.UserID    `schema:"author_id,omitempty"`
	Mentions  discord.UserID    `schema:"mentions,omitempty"`
	MaxID     discord.MessageID `schema:"max_id,omitempty"`
	MinID     discord.MessageID `schema:"min_id,omitempty"`
}

type SearchResponse struct {
	AnalyticsID  string              `json:"analytics_id"`
	Messages     [][]discord.Message `json:"messages"`
	TotalResults uint                `json:"total_results"`
}

// Search searches through a guild's messages. It only works for user accounts.
func (c *Client) Search(guildID discord.GuildID, data SearchData) (SearchResponse, error) {
	var resp SearchResponse

	return resp, c.RequestJSON(
		&resp, "GET",
		EndpointGuilds+guildID.String()+"/messages/search",
		httputil.WithSchema(c, data),
	)
}
