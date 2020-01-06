package api

import "github.com/diamondburned/arikawa/discord"

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

// Guilds returns maximum 100 of your guilds. To paginate, call MyGuildsRange.
// Guilds returned have some fields filled only (ID, Name, Icon, Owner,
// Permissions).
func (c *Client) Guilds() ([]discord.Guild, error) {
	var gs []discord.Guild
	return gs, c.RequestJSON(&gs, "GET", EndpointMe+"/guilds")
}

// func (c *Client) GuildsRange()

// func (c *Client)
