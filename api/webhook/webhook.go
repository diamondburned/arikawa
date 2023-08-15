// Package webhook provides means to interact with webhooks directly and not
// through the bot API.
package webhook

import (
	"context"
	"mime/multipart"
	"net/url"
	"regexp"
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

var webhookURLRe = regexp.MustCompile(`https://discord(?:app)?.com/api/webhooks/(\d+)/(.+)`)

// ParseURL parses the given Discord webhook URL.
func ParseURL(webhookURL string) (id discord.WebhookID, token string, err error) {
	matches := webhookURLRe.FindStringSubmatch(webhookURL)
	if matches == nil {
		return 0, "", errors.New("invalid webhook URL")
	}

	idInt, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, "", errors.Wrap(err, "failed to parse webhook ID")
	}

	return discord.WebhookID(idInt), matches[2], nil
}

// Session keeps a single webhook session. It is referenced by other webhook
// clients using the same session.
type Session struct {
	// Limiter is the rate limiter used for the client. This field should not be
	// changed, as doing so is potentially racy.
	Limiter *rate.Limiter

	// ID is the ID of the webhook.
	ID discord.WebhookID
	// Token is the token of the webhook.
	Token string
}

// OnRequest should be called on each client request to inject itself.
func (s *Session) OnRequest(r httpdriver.Request) error {
	return s.Limiter.Acquire(r.GetContext(), r.GetPath())
}

// OnResponse should be called after each client request to clean itself up.
func (s *Session) OnResponse(r httpdriver.Request, resp httpdriver.Response) error {
	return s.Limiter.Release(r.GetPath(), httpdriver.OptHeader(resp))
}

// Client creates a new Webhook API client from the session.
func (s *Session) Client() *Client {
	return &Client{httputil.NewClient(), s}
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

// NewFromURL creates a new webhook client using the passed webhook URL. It
// uses its own rate limiter.
func NewFromURL(url string) (*Client, error) {
	id, token, err := ParseURL(url)
	if err != nil {
		return nil, err
	}
	return New(id, token), nil
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
	// Content are the message contents (up to 2000 characters).
	//
	// Required: one of content, file, embeds
	Content string `json:"content,omitempty"`

	// ThreadID causes the message to be sent to the specified thread within
	// the webhook's channel. The thread will automatically be unarchived.
	ThreadID discord.CommandID `json:"-"`

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

	// Components is the list of components (such as buttons) to be attached to
	// the message.
	Components discord.ContainerComponents `json:"components,omitempty"`

	// Files represents a list of files to upload. This will not be
	// JSON-encoded and will only be available through WriteMultipart.
	Files []sendpart.File `json:"-"`

	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
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

	sum := 0
	for i, embed := range data.Embeds {
		if err := embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "embed error at "+strconv.Itoa(i))
		}
		sum += embed.Length()
		if sum > 6000 {
			return nil, &discord.OverboundError{sum, 6000, "sum of all text in embeds"}
		}
	}

	param := make(url.Values, 2)
	if wait {
		param["wait"] = []string{"true"}
	}
	if data.ThreadID.IsValid() {
		param["thread_id"] = []string{data.ThreadID.String()}
	}

	var URL = api.EndpointWebhooks + c.ID.String() + "/" + c.Token + "?" + param.Encode()

	var msg *discord.Message
	var ptr interface{}
	if wait {
		ptr = &msg
	}

	return msg, sendpart.POST(c.Client, data, ptr, URL)
}

// Message returns a previously-sent webhook message from the same token.
func (c *Client) Message(messageID discord.MessageID) (*discord.Message, error) {
	var m *discord.Message
	return m, c.RequestJSON(
		&m, "GET",
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token+"/messages/"+messageID.String())
}

// https://discord.com/developers/docs/resources/webhook#edit-webhook-message-jsonform-params
type EditMessageData struct {
	// Content is the new message contents (up to 2000 characters).
	Content option.NullableString `json:"content,omitempty"`
	// Embeds contains embedded rich content.
	Embeds *[]discord.Embed `json:"embeds,omitempty"`
	// Components contains the new components to attach.
	Components *discord.ContainerComponents `json:"components,omitempty"`
	// AllowedMentions are the allowed mentions for a message.
	AllowedMentions *api.AllowedMentions `json:"allowed_mentions,omitempty"`
	// Attachments are the attached files to keep
	Attachments *[]discord.Attachment `json:"attachments,omitempty"`

	Files []sendpart.File `json:"-"`
}

// EditMessage edits a previously-sent webhook message from the same webhook.
func (c *Client) EditMessage(messageID discord.MessageID, data EditMessageData) (*discord.Message, error) {
	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}
	if data.Embeds != nil {
		sum := 0
		for _, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{sum, 6000, "sum of text in embeds"}
			}
		}
	}
	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token+"/messages/"+messageID.String())
}

// NeedsMultipart returns true if the SendMessageData has files.
func (data EditMessageData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

func (data EditMessageData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
}

// DeleteMessage deletes a message that was previously created by the same
// webhook.
func (c *Client) DeleteMessage(messageID discord.MessageID) error {
	return c.FastRequest("DELETE",
		api.EndpointWebhooks+c.ID.String()+"/"+c.Token+"/messages/"+messageID.String())
}
