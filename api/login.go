package api

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

var (
	EndpointAuth  = Endpoint + "auth/"
	EndpointLogin = EndpointAuth + "login"

	EndpointMFA  = EndpointAuth + "mfa/"
	EndpointTOTP = EndpointMFA + "totp"
	EndpointSMS  = EndpointMFA + "sms"
)

type (
	LoginSettings struct {
		Locale discord.Language `json:"locale"`
		Theme  string           `json:"theme"`
	}

	LoginResponse struct {
		UserID          discord.UserID `json:"user_id"`
		Token           string         `json:"token"`
		UserSettings    LoginSettings  `json:"user_settings"`
		RequiredActions []string       `json:"required_actions"`

		Ticket          string `json:"ticket"`
		LoginInstanceID string `json:"login_instance_id"`
		MFA             bool   `json:"mfa"`
		TOTP            bool   `json:"totp"`
		SMS             bool   `json:"sms"`
		Backup          bool   `json:"backup"`
	}
)

// Login retrieves an authentication token for the given credentials.
// login is the user's email or E.164-formatted phone number
func (c *Client) Login(login, password string) (*LoginResponse, error) {
	var param struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	param.Login = login
	param.Password = password

	var r *LoginResponse
	return r, c.RequestJSON(&r, "POST", EndpointLogin, httputil.WithJSONBody(param))
}

// SendMFASMS sends a multi-factor authentication code to the user's phone number for verification.
// Returns the redacted phone number the SMS was sent to.
func (c *Client) SendMFASMS(ticket string) (string, error) {
	body := struct {
		Ticket string `json:"ticket"`
	}{ticket}
	var r struct {
		Phone string `json:"phone"`
	}
	return r.Phone, c.RequestJSON(&r, "POST", EndpointSMS+"/send", httputil.WithJSONBody(body))
}

// TOTP verifies a multi-factor login using the TOTP code or backup code and retrieves an authentication token using the specified authenticator type.
func (c *Client) TOTP(code, ticket, loginInstanceID string) (*LoginResponse, error) {
	return c.mfa(EndpointTOTP, code, ticket, loginInstanceID)
}

// SMS verifies a multi-factor login using the code sent to the user's phone number via SMS and retrieves an authentication token.
func (c *Client) SMS(code, ticket, loginInstanceID string) (*LoginResponse, error) {
	return c.mfa(EndpointSMS, code, ticket, loginInstanceID)
}

func (c *Client) mfa(endpoint, code, ticket, loginInstanceID string) (*LoginResponse, error) {
	body := struct {
		Code            string `json:"code"`
		Ticket          string `json:"ticket"`
		LoginInstanceID string `json:"login_instance_id"`
	}{Code: code, Ticket: ticket, LoginInstanceID: loginInstanceID}
	var r *LoginResponse
	return r, c.RequestJSON(&r, "POST", endpoint, httputil.WithJSONBody(body))
}
