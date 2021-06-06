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

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	var param struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	param.Email = email
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
