package api

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json/option"
)

var (
	EndpointUsers = Endpoint + "users/"
	EndpointMe    = EndpointUsers + "@me"
)

// User returns a user object for a given user ID.
func (c *Client) User(userID discord.Snowflake) (*discord.User, error) {
	var u *discord.User
	return u, c.RequestJSON(&u, "GET", EndpointUsers+userID.String())
}

// Me returns the user object of the requester's account. For OAuth2, this
// requires the identify scope, which will return the object without an email,
// and optionally the email scope, which returns the object with an email.
func (c *Client) Me() (*discord.User, error) {
	var me *discord.User
	return me, c.RequestJSON(&me, "GET", EndpointMe)
}

// https://discord.com/developers/docs/resources/user#modify-current-user-json-params
type ModifySelfData struct {
	// Username is the user's username, if changed may cause the user's
	// discriminator to be randomized.
	Username option.String `json:"username,omitempty"`
	// Avatar modifies the user's avatar.
	Avatar *Image `json:"image,omitempty"`
}

// ModifyMe modifies the requester's user account settings.
func (c *Client) ModifyMe(data ModifySelfData) (*discord.User, error) {
	var u *discord.User
	return u, c.RequestJSON(&u, "PATCH", EndpointMe, httputil.WithJSONBody(data))
}

// PrivateChannels returns a list of DM channel objects. For bots, this is no
// longer a supported method of getting recent DMs, and will return an empty
// array.
func (c *Client) PrivateChannels() ([]discord.Channel, error) {
	var dms []discord.Channel
	return dms, c.RequestJSON(&dms, "GET", EndpointMe+"/channels")
}

// CreatePrivateChannel creates a new DM channel with a user.
func (c *Client) CreatePrivateChannel(recipientID discord.Snowflake) (*discord.Channel, error) {
	var param struct {
		RecipientID discord.Snowflake `json:"recipient_id"`
	}

	param.RecipientID = recipientID

	var dm *discord.Channel
	return dm, c.RequestJSON(&dm, "POST", EndpointMe+"/channels", httputil.WithJSONBody(param))
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
