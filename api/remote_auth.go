package api

import "github.com/diamondburned/arikawa/v3/utils/httputil"

var EndpointRemoteAuthLogin = EndpointMe + "/remote-auth/login"

// ExchangeRemoteAuthTicket exchanges a remote auth ticket for an authentication token.
// The token must be decrypted using the client's private key.
// Returns the authentication token encrypted with the client's public key.
func (c *Client) ExchangeRemoteAuthTicket(ticket string) (string, error) {
	body := struct {
		Ticket string `json:"ticket"`
	}{ticket}
	var resp struct {
		EncryptedToken string `json:"encrypted_token"`
	}
	return resp.EncryptedToken, c.RequestJSON(&resp, "POST", EndpointRemoteAuthLogin, httputil.WithJSONBody(body))
}
