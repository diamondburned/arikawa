package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/httputil"
)

const EndpointUsers = Endpoint + "users/"
const EndpointMe = EndpointUsers + "@me"

func (c *Client) User(userID discord.Snowflake) (*discord.User, error) {
	var u *discord.User
	return u, c.RequestJSON(&u, "GET",
		EndpointUsers+userID.String())
}

func (c *Client) Me() (*discord.User, error) {
	var me *discord.User
	return me, c.RequestJSON(&me, "GET", EndpointMe)
}

type ModifySelfData struct {
	Username string `json:"username,omitempty"`
	Avatar   Image  `json:"image,omitempty"`
}

func (c *Client) ModifyMe(data ModifySelfData) (*discord.User, error) {
	var u *discord.User
	return u, c.RequestJSON(&u, "PATCH", EndpointMe)
}

// Guilds returns all guilds, automatically paginating. Be careful, as this
// method may abuse the API by requesting thousands or millions of guilds. For
// lower-level access, usee GuildsRange. Guilds returned have some fields
// filled only (ID, Name, Icon, Owner, Permissions).
func (c *Client) Guilds(max uint) ([]discord.Guild, error) {
	var guilds []discord.Guild
	var after discord.Snowflake = 0

	const hardLimit int = 100

	for fetch := uint(hardLimit); max > 0; fetch = uint(hardLimit) {
		if fetch > max {
			fetch = max
		}
		max -= fetch

		g, err := c.GuildsAfter(after, fetch)
		if err != nil {
			return guilds, err
		}
		guilds = append(guilds, g...)

		if len(g) < hardLimit {
			break
		}

		after = g[hardLimit-1].ID
	}

	return guilds, nil
}

// GuildsBefore fetches guilds. Check GuildsRange.
func (c *Client) GuildsBefore(
	before discord.Snowflake, limit uint) ([]discord.Guild, error) {

	return c.GuildsRange(before, 0, limit)
}

// GuildsAfter fetches guilds. Check GuildsRange.
func (c *Client) GuildsAfter(
	after discord.Snowflake, limit uint) ([]discord.Guild, error) {

	return c.GuildsRange(0, after, limit)
}

// GuildsRange fetches guilds. The limit is 1-100.
func (c *Client) GuildsRange(
	before, after discord.Snowflake, limit uint) ([]discord.Guild, error) {

	if limit == 0 {
		limit = 100
	}

	if limit > 100 {
		limit = 100
	}

	var param struct {
		Before discord.Snowflake `schema:"before"`
		After  discord.Snowflake `schema:"after"`

		Limit uint `schema:"limit"`
	}

	var gs []discord.Guild
	return gs, c.RequestJSON(
		&gs, "GET",
		EndpointMe+"/guilds",
		httputil.WithSchema(c, param),
	)
}

func (c *Client) LeaveGuild(guildID discord.Snowflake) error {
	return c.FastRequest("DELETE", EndpointMe+"/guilds/"+guildID.String())
}

func (c *Client) PrivateChannels() ([]discord.Channel, error) {
	var dms []discord.Channel
	return dms, c.RequestJSON(&dms, "GET", EndpointMe+"/channels")
}

func (c *Client) CreatePrivateChannel(
	recipient discord.Snowflake) (*discord.Channel, error) {

	var param struct {
		RecipientID discord.Snowflake `json:"recipient_id"`
	}

	param.RecipientID = recipient

	var dm *discord.Channel
	return dm, c.RequestJSON(&dm, "POST", EndpointMe+"/channels",
		httputil.WithJSONBody(c, param))
}

// shitty SDK, don't care, PR welcomed
// func (c *Client) CreateGroup(tokens []string, nicks map[])

// func (c *Client) UserConnections() ([]discord.Connection, error) {}
