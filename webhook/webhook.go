// Package webhook provides means to interact with webhooks directly and not
// through the bot API.
package webhook

import (
	"mime/multipart"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json"
)

// DefaultHTTPClient is the httputil.Client used in the helper methods.
var DefaultHTTPClient = httputil.NewClient()

// Client is the client used to interact with a webhook.
type Client struct {
	// Client is the httputil.Client used to call Discord's API.
	*httputil.Client
	// ID is the id of the webhook.
	ID discord.WebhookID
	// Token is the token of the webhook.
	Token string
}

// NewClient creates a new Client using the passed token and id.
func NewClient(id discord.WebhookID, token string) *Client {
	return NewCustomClient(id, token, httputil.NewClient())
}

// NewCustomClient creates a new Client creates a new Client using the passed
// token and id and makes API calls using the passed httputil.Client
func NewCustomClient(id discord.WebhookID, token string, c *httputil.Client) *Client {
	return &Client{
		Client: c,
		ID:     id,
		Token:  token,
	}
}

// Get gets the webhook.
func (c *Client) Get() (*discord.Webhook, error) {
	var w *discord.Webhook
	return w, c.RequestJSON(&w, "GET", api.EndpointWebhooks+c.ID.String()+"/"+c.Token)
}

// Modify modifies the webhook.
func (c *Client) Modify(data api.ModifyWebhookData) (*discord.Webhook, error) {
	var w *discord.Webhook
	return w, c.RequestJSON(
		&w, "PATCH",
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token,
		httputil.WithJSONBody(data),
	)
}

// Delete deletes a webhook permanently.
func (c *Client) Delete() error {
	return c.FastRequest("DELETE", api.EndpointWebhooks+c.ID.String()+"/"+c.Token)
}

// Execute sends a message to the webhook, but doesn't wait for the message to
// get created. This is generally faster, but only applicable if no further
// interaction is required.
func (c *Client) Execute(data api.ExecuteWebhookData) (err error) {
	_, err = c.execute(data, false)
	return
}

// ExecuteAndWait executes the webhook, and waits for the generated
// discord.Message to be returned.
func (c *Client) ExecuteAndWait(data api.ExecuteWebhookData) (*discord.Message, error) {
	return c.execute(data, true)
}

func (c *Client) execute(data api.ExecuteWebhookData, wait bool) (*discord.Message, error) {
	if data.Content == "" && len(data.Embeds) == 0 && len(data.Files) == 0 {
		return nil, api.ErrEmptyMessage
	}

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	for i, embed := range data.Embeds {
		if err := embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "embed error at "+strconv.Itoa(i))
		}
	}

	var param = url.Values{}
	if wait {
		param.Set("wait", "true")
	}

	var URL = api.EndpointWebhooks + c.ID.String() + "/" + c.Token + "?" + param.Encode()
	var msg *discord.Message

	if len(data.Files) == 0 {
		// No files, so no need for streaming.
		return msg, c.RequestJSON(&msg, "POST", URL,
			httputil.WithJSONBody(data))
	}

	writer := func(mw *multipart.Writer) error {
		return data.WriteMultipart(mw)
	}

	resp, err := c.MeanwhileMultipart(writer, "POST", URL)
	if err != nil {
		return nil, err
	}

	var body = resp.GetBody()
	defer body.Close()

	if !wait {
		// Since we didn't tell Discord to wait, we have nothing to parse.
		return nil, nil
	}

	return msg, json.DecodeStream(body, &msg)
}

// Get is a shortcut for NewCustomClient(token, id, DefaultHTTPClient).Get().
func Get(id discord.WebhookID, token string) (*discord.Webhook, error) {
	return NewCustomClient(id, token, DefaultHTTPClient).Get()
}

// Modify is a shortcut for
// NewCustomClient(token, id, DefaultHTTPClient).Modify(data).
func Modify(
	id discord.WebhookID, token string, data api.ModifyWebhookData) (*discord.Webhook, error) {

	return NewCustomClient(id, token, DefaultHTTPClient).Modify(data)
}

// Delete is a shortcut for
// NewCustomClient(token, id, DefaultHTTPClient).Delete().
func Delete(id discord.WebhookID, token string) error {
	return NewCustomClient(id, token, DefaultHTTPClient).Delete()
}

// Execute is a shortcut for
// NewCustomClient(token, id, DefaultHTTPClient).Execute(data).
func Execute(id discord.WebhookID, token string, data api.ExecuteWebhookData) error {
	return NewCustomClient(id, token, DefaultHTTPClient).Execute(data)
}

// ExecuteAndWait is a shortcut for
// NewCustomClient(token, id, DefaultHTTPClient).ExecuteAndWait(data).
func ExecuteAndWait(
	id discord.WebhookID, token string, data api.ExecuteWebhookData) (*discord.Message, error) {

	return NewCustomClient(id, token, DefaultHTTPClient).ExecuteAndWait(data)
}
