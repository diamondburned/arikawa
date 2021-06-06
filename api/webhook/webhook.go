// Package webhook provides means to interact with webhooks directly and not
// through the bot API.
package webhook

import (
	"context"
	"mime/multipart"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/rate"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

// TODO: if there's ever an Arikawa v3, then a new Client abstraction could be
// made that wraps around Session being an interface. Just a food for thought.

// Session keeps a single webhook session. It is referenced by other webhook
// clients using the same session.
type Session struct {
	// Limiter is the rate limiter used for the client. This field should not be
	// changed, as doing so is potentially racy.
	Limiter *rate.Limiter
	// Token is the token of the webhook.
	Token string
	// ID is the ID of the webhook.
	ID discord.WebhookID
}

// OnRequest should be called on each client request to inject itself.
func (s *Session) OnRequest(r httpdriver.Request) error {
	return s.Limiter.Acquire(r.GetContext(), r.GetPath())
}

// OnResponse should be called after each client request to clean itself up.
func (s *Session) OnResponse(r httpdriver.Request, resp httpdriver.Response) error {
	return s.Limiter.Release(r.GetPath(), httpdriver.OptHeader(resp))
}

// Client is the client used to interact with a webhook.
type Client struct {
	// Client is the httputil.Client used to call Discord's API.
	*httputil.Client
	*Session
}

// New creates a new Client using the passed webhook token and ID. It uses its
// own rate limiter.
func New(id discord.WebhookID, token string) *Client {
	return NewCustom(id, token, httputil.NewClient())
}

// NewCustom creates a new webhook client using the passed webhook token, ID and
// a copy of the given httputil.Client. The copy will have a new rate limiter
// added in.
func NewCustom(id discord.WebhookID, token string, hcl *httputil.Client) *Client {
	ses := Session{
		Limiter: rate.NewLimiter(api.Path),
		ID:      id,
		Token:   token,
	}

	hcl = hcl.Copy()
	hcl.OnRequest = append(hcl.OnRequest, ses.OnRequest)
	hcl.OnResponse = append(hcl.OnResponse, ses.OnResponse)

	return &Client{
		Client:  hcl,
		Session: &ses,
	}
}

// FromAPI creates a new client that shares the same internal HTTP client with
// the one in the API's. This is often useful for bots that need webhook
// interaction, since the rate limiter is shared.
func FromAPI(id discord.WebhookID, token string, c *api.Client) *Client {
	return &Client{
		Client: c.Client,
		Session: &Session{
			Limiter: c.Limiter,
			ID:      id,
			Token:   token,
		},
	}
}

// WithContext returns a shallow copy of Client with the given context. It's
// used for method timeouts and such. This method is thread-safe.
func (c *Client) WithContext(ctx context.Context) *Client {
	return &Client{
		Client:  c.Client.WithContext(ctx),
		Session: c.Session,
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
type ExecuteData struct {
	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
	// Username overrides the default username of the webhook
	Username string `json:"username,omitempty"`
	// AvatarURL overrides the default avatar of the webhook.
	AvatarURL discord.URL `json:"avatar_url,omitempty"`
	// Content are the message contents (up to 2000 characters).
	//
	// Required: one of content, file, embeds
	Content string `json:"content,omitempty"`
	// Embeds contains embedded rich content.
	//
	// Required: one of content, file, embeds
	Embeds []discord.Embed `json:"embeds,omitempty"`
	// Files represents a list of files to upload. This will not be JSON-encoded
	// and will only be available through WriteMultipart.
	Files []sendpart.File `json:"-"`
	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
}

// NeedsMultipart returns true if the ExecuteWebhookData has files.
func (data ExecuteData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

// WriteMultipart writes the webhook data into the given multipart body. It does
// not close body.
func (data ExecuteData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// Execute sends a message to the webhook, but doesn't wait for the message to
// get created. This is generally faster, but only applicable if no further
// interaction is required.
func (c *Client) Execute(data ExecuteData) (err error) {
	_, err = c.execute(data, false)
	return
}

// ExecuteAndWait executes the webhook, and waits for the generated
// discord.Message to be returned.
func (c *Client) ExecuteAndWait(data ExecuteData) (*discord.Message, error) {
	return c.execute(data, true)
}

func (c *Client) execute(data ExecuteData, wait bool) (*discord.Message, error) {
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

	URL := api.EndpointWebhooks + c.ID.String() + "/" + c.Token + "?" + param.Encode()

	var msg *discord.Message
	var ptr interface{}
	if wait {
		ptr = &msg
	}

	return msg, sendpart.POST(c.Client, data, ptr, URL)
}

// https://discord.com/developers/docs/resources/webhook#edit-webhook-message-jsonform-params
type EditMessageData struct {
	// Content are the message contents. They may be up to 2000 characters
	// characters long.
	Content option.NullableString `json:"content,omitempty"`
	// Embeds is an array of up to 10 discord.Embeds.
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// AllowedMentions are the AllowedMentions for the message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
}

// EditMessage edits a previously-sent webhook message from the same webhook.
func (c *Client) EditMessage(messageID discord.MessageID, data EditMessageData) error {
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
