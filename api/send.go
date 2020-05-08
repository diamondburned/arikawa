package api

import (
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/pkg/errors"
)

const AttachmentSpoilerPrefix = "SPOILER_"

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

// AllowedMentions is a whitelist of mentions for a message.
// https://discordapp.com/developers/docs/resources/channel#allowed-mentions-object
//
// Whitelists
//
// Roles and Users are slices that act as whitelists for IDs that are allowed to
// be mentioned. For example, if only 1 ID is provided in Users, then only that
// ID will be parsed in the message. No other IDs will be. The same example also
// applies for roles.
//
// If Parse is an empty slice and both Users and Roles are empty slices, then no
// mentions will be parsed.
//
// Constraints
//
// If the Users slice is not empty, then Parse must not have AllowUserMention.
// Likewise, if the Roles slice is not empty, then Parse must not have
// AllowRoleMention. This is because everything provided in Parse will make
// Discord parse it completely, meaning they would be mutually exclusive with
// whitelist slices, Roles and Users.
type AllowedMentions struct {
	Parse []AllowedMentionType `json:"parse"`
	Roles []discord.Snowflake  `json:"roles,omitempty"` // max 100
	Users []discord.Snowflake  `json:"users,omitempty"` // max 100
}

// AllowedMentionType is a constant that tells Discord what is allowed to parse
// from a message content. This can help prevent things such as an unintentional
// @everyone mention.
type AllowedMentionType string

const (
	// AllowRoleMention makes Discord parse roles in the content.
	AllowRoleMention AllowedMentionType = "roles"
	// AllowUserMention makes Discord parse user mentions in the content.
	AllowUserMention AllowedMentionType = "users"
	// AllowEveryoneMention makes Discord parse @everyone mentions.
	AllowEveryoneMention AllowedMentionType = "everyone"
)

// Verify checks the AllowedMentions against constraints mentioned in
// AllowedMentions' documentation. This will be called on SendMessageComplex.
func (am AllowedMentions) Verify() error {
	if len(am.Roles) > 100 {
		return errors.Errorf("Roles slice length %d is over 100", len(am.Roles))
	}
	if len(am.Users) > 100 {
		return errors.Errorf("Users slice length %d is over 100", len(am.Users))
	}

	for _, allowed := range am.Parse {
		switch allowed {
		case AllowRoleMention:
			if len(am.Roles) > 0 {
				return errors.New(`Parse has AllowRoleMention and Roles slice is not empty`)
			}
		case AllowUserMention:
			if len(am.Users) > 0 {
				return errors.New(`Parse has AllowUserMention and Users slice is not empty`)
			}
		}
	}

	return nil
}

// ErrEmptyMessage is returned if either a SendMessageData or an
// ExecuteWebhookData has both an empty Content and no Embed(s).
var ErrEmptyMessage = errors.New("Message is empty")

// SendMessageFile represents a file to be uploaded to Discord.
type SendMessageFile struct {
	Name   string
	Reader io.Reader
}

// SendMessageData is the full structure to send a new message to Discord with.
type SendMessageData struct {
	// Either of these fields must not be empty.
	Content string `json:"content,omitempty"`
	Nonce   string `json:"nonce,omitempty"`

	TTS   bool           `json:"tts,omitempty"`
	Embed *discord.Embed `json:"embed,omitempty"`

	Files []SendMessageFile `json:"-"`

	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
}

func (data *SendMessageData) WriteMultipart(body *multipart.Writer) error {
	return writeMultipart(body, data, data.Files)
}

func (c *Client) SendMessageComplex(
	channelID discord.Snowflake, data SendMessageData) (*discord.Message, error) {

	if data.Content == "" && data.Embed == nil && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
	}

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "AllowedMentions error")
		}
	}

	if data.Embed != nil {
		if err := data.Embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "Embed error")
		}
	}

	var URL = EndpointChannels + channelID.String() + "/messages"
	var msg *discord.Message

	if len(data.Files) == 0 {
		// No files, so no need for streaming.
		return msg, c.RequestJSON(&msg, "POST", URL, httputil.WithJSONBody(data))
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

	return msg, json.DecodeStream(body, &msg)
}

type ExecuteWebhookData struct {
	// Either of these fields must not be empty.
	Content string `json:"content,omitempty"`
	Nonce   string `json:"nonce,omitempty"`

	TTS    bool            `json:"tts,omitempty"`
	Embeds []discord.Embed `json:"embeds,omitempty"`

	Files []SendMessageFile `json:"-"`

	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`

	// Optional fields specific to Webhooks.
	Username  string      `json:"username,omitempty"`
	AvatarURL discord.URL `json:"avatar_url,omitempty"`
}

func (data *ExecuteWebhookData) WriteMultipart(body *multipart.Writer) error {
	return writeMultipart(body, data, data.Files)
}

// ExecuteWebhook sends a message to the webhook. If wait is bool, Discord will
// wait for the message to be delivered and will return the message body. This
// also means the returned message will only be there if wait is true.
func (c *Client) ExecuteWebhook(
	webhookID discord.Snowflake,
	token string,
	wait bool, // if false, then nil returned for *Message.
	data ExecuteWebhookData) (*discord.Message, error) {

	if data.Content == "" && len(data.Embeds) == 0 && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
	}

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "AllowedMentions error")
		}
	}

	for i, embed := range data.Embeds {
		if err := embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "Embed error at "+strconv.Itoa(i))
		}
	}

	var param = url.Values{}
	if wait {
		param.Set("wait", "true")
	}

	var URL = EndpointWebhooks + webhookID.String() + "/" + token + "?" + param.Encode()
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

func writeMultipart(body *multipart.Writer, item interface{}, files []SendMessageFile) error {
	defer body.Close()

	// Encode the JSON body first
	w, err := body.CreateFormField("payload_json")
	if err != nil {
		return errors.Wrap(err, "Failed to create bodypart for JSON")
	}

	if err := json.EncodeStream(w, item); err != nil {
		return errors.Wrap(err, "Failed to encode JSON")
	}

	for i, file := range files {
		num := strconv.Itoa(i)

		w, err := body.CreateFormFile("file"+num, file.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to create bodypart for "+num)
		}

		if _, err := io.Copy(w, file.Reader); err != nil {
			return errors.Wrap(err, "Failed to write for file "+num)
		}
	}

	return nil
}
