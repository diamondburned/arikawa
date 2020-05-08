package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
)

var (
	EndpointUsers = Endpoint + "users/"
	EndpointMe    = EndpointUsers + "@me"
)

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
		httputil.WithJSONBody(param))
}

// ChangeOwnNickname only replies with the nickname back, so we're not even
// going to bother.
func (c *Client) ChangeOwnNickname(
	guildID discord.Snowflake, nick string) error {

	var param struct {
		Nick string `json:"nick"`
	}

	param.Nick = nick

	return c.FastRequest(
		"PATCH",
		EndpointGuilds+guildID.String()+"/members/@me/nick",
		httputil.WithJSONBody(param),
	)
}

// shitty SDK, don't care, PR welcomed
// func (c *Client) CreateGroup(tokens []string, nicks map[])

// func (c *Client) UserConnections() ([]discord.Connection, error) {}
