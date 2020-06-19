package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json/option"
)

type SearchData struct {
	Content   option.String     `schema:"content,omitempty"`
	AuthorID  discord.Snowflake `schema:"author_id,omitempty"`
	Mentions  discord.Snowflake `schema:"mentions,omitempty"`
	Has       option.String     `schema:"has,omitempty"`
	MaxID     discord.Snowflake `schema:"max_id,omitempty"`
	MinID     discord.Snowflake `schema:"min_id,omitempty"`
	ChannelID discord.Snowflake `schema:"channel_id,omitempty"`
}

type SearchResponse struct {
	AnalyticsID  string              `json:"analytics_id"`
	Messages     [][]discord.Message `json:"messages"`
	TotalResults uint                `json:"total_results"`
}

func (c *Client) Search(guildID discord.Snowflake, data SearchData) (SearchResponse, error) {
	var resp SearchResponse

	return resp, c.RequestJSON(
		&resp, "GET",
		EndpointGuilds+guildID.String()+"/messages/search",
		httputil.WithSchema(c, data),
	)
}
