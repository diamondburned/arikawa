package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var (
	EndpointUsers = Endpoint + "users/"
	EndpointMe    = EndpointUsers + "@me"
)

// User returns a user object for a given user ID.
func (c *Client) User(userID discord.UserID) (*discord.User, error) {
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

	AuditLogReason `json:"-"`
}

// ModifyMe modifies the requester's user account settings.
func (c *Client) ModifyMe(data ModifySelfData) (*discord.User, error) {
	var u *discord.User
	return u, c.RequestJSON(
		&u,
		"PATCH", EndpointMe,
		httputil.WithJSONBody(data), httputil.WithHeaders(data.Header()),
	)
}

// ChangeOwnNickname modifies the nickname of the current user in a guild.
//
// Fires a Guild Member Update Gateway event.
func (c *Client) ChangeOwnNickname(
	guildID discord.GuildID, nick string) error {

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

// PrivateChannels returns a list of DM channel objects. For bots, this is no
// longer a supported method of getting recent DMs, and will return an empty
// array.
func (c *Client) PrivateChannels() ([]discord.Channel, error) {
	var dms []discord.Channel
	return dms, c.RequestJSON(&dms, "GET", EndpointMe+"/channels")
}

// CreatePrivateChannel creates a new DM channel with a user.
func (c *Client) CreatePrivateChannel(recipientID discord.UserID) (*discord.Channel, error) {
	var param struct {
		RecipientID discord.UserID `json:"recipient_id"`
	}

	param.RecipientID = recipientID

	var dm *discord.Channel
	return dm, c.RequestJSON(&dm, "POST", EndpointMe+"/channels", httputil.WithJSONBody(param))
}

// UserConnections returns a list of connection objects. Requires the
// connections OAuth2 scope.
func (c *Client) UserConnections() ([]discord.Connection, error) {
	var conn []discord.Connection
	return conn, c.RequestJSON(&conn, "GET", EndpointMe+"/connections")
}

// Note gets the note for the given user. This endpoint is undocumented and
// might only work for user accounts.
func (c *Client) Note(userID discord.UserID) (string, error) {
	var body struct {
		Note string `json:"note"`
	}

	return body.Note, c.RequestJSON(&body, "GET", EndpointMe+"/notes/"+userID.String())
}

// SetNote sets a note for the user. This endpoint is undocumented and might
// only work for user accounts.
func (c *Client) SetNote(userID discord.UserID, note string) error {
	var body = struct {
		Note string `json:"note"`
	}{
		Note: note,
	}

	return c.FastRequest(
		"PUT", EndpointMe+"/notes/"+userID.String(),
		httputil.WithJSONBody(body),
	)
}

// SetRelationship sets the relationship type between the current user and the
// given user.
func (c *Client) SetRelationship(userID discord.UserID, t discord.RelationshipType) error {
	var body = struct {
		Type discord.RelationshipType `json:"type"`
	}{
		Type: t,
	}

	return c.FastRequest(
		"PUT", EndpointMe+"/relationships/"+userID.String(),
		httputil.WithJSONBody(body),
	)
}

// DeleteRelationship deletes the relationship between the current user and the
// given user.
func (c *Client) DeleteRelationship(userID discord.UserID) error {
	return c.FastRequest("DELETE", EndpointMe+"/relationships/"+userID.String())
}
