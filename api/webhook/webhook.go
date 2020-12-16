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
	"github.com/diamondburned/arikawa/v2/utils/json/option"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
)

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

// https://discord.com/developers/docs/resources/webhook#execute-webhook-jsonform-params
type ExecuteWebhookData struct {
	// Content are the message contents (up to 2000 characters).
	//
	// Required: one of content, file, embeds
	Content string `json:"content,omitempty"`

	// Username overrides the default username of the webhook
	Username string `json:"username,omitempty"`
	// AvatarURL overrides the default avatar of the webhook.
	AvatarURL discord.URL `json:"avatar_url,omitempty"`

	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
	// Embeds contains embedded rich content.
	//
	// Required: one of content, file, embeds
	Embeds []discord.Embed `json:"embeds,omitempty"`

	// Files represents a list of files to upload. This will not be JSON-encoded
	// and will only be available through WriteMultipart.
	Files []sendpart.File `json:"-"`

	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
}

// NeedsMultipart returns true if the ExecuteWebhookData has files.
func (data ExecuteWebhookData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

// WriteMultipart writes the webhook data into the given multipart body. It does
// not close body.
func (data ExecuteWebhookData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// Execute sends a message to the webhook, but doesn't wait for the message to
// get created. This is generally faster, but only applicable if no further
// interaction is required.
func (c *Client) Execute(data ExecuteWebhookData) (err error) {
	_, err = c.execute(data, false)
	return
}

// ExecuteAndWait executes the webhook, and waits for the generated
// discord.Message to be returned.
func (c *Client) ExecuteAndWait(data ExecuteWebhookData) (*discord.Message, error) {
	return c.execute(data, true)
}

func (c *Client) execute(data ExecuteWebhookData, wait bool) (*discord.Message, error) {
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

	var param url.Values
	if wait {
		param = url.Values{"wait": {"true"}}
	}

	var URL = api.EndpointWebhooks + c.ID.String() + "/" + c.Token + "?" + param.Encode()

	var msg *discord.Message
	var ptr interface{}
	if wait {
		ptr = &msg
	}

	return msg, sendpart.POST(c.Client, data, ptr, URL)
}

// https://discord.com/developers/docs/resources/webhook#edit-webhook-message-jsonform-params
type EditWebhookMessageData struct {
	// Content are the message contents. They may be up to 2000 characters
	// characters long.
	Content option.NullableString `json:"content,omitempty"`
	// Embeds is an array of up to 10 discord.Embeds.
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// AllowedMentions are the AllowedMentions for the message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
}

// EditMessage edits a previously-sent webhook message from the same webhook.
func (c *Client) EditMessage(messageID discord.MessageID, data EditWebhookMessageData) error {
	return c.FastRequest("PATCH",
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token+"/messages/"+messageID.String(),
		httputil.WithJSONBody(data))
}

// DeleteMessage deletes a message that was previously created by the same
// webhook.
func (c *Client) DeleteMessage(messageID discord.MessageID) error {
	return c.FastRequest("DELETE",
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token+"/messages/"+messageID.String())
}
