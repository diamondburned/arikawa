package api

import "github.com/diamondburned/arikawa/v3/utils/httputil"

var (
	EndpointAuth  = Endpoint + "auth/"
	EndpointLogin = EndpointAuth + "login"
	EndpointTOTP  = EndpointAuth + "mfa/totp"
)

type LoginResponse struct {
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
}

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

func (c *Client) TOTP(code, ticket string) (*LoginResponse, error) {
	var param struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}
	param.Code = code
	param.Ticket = ticket

	var r *LoginResponse
	return r, c.RequestJSON(&r, "POST", EndpointTOTP, httputil.WithJSONBody(param))
}
