package api

import (
	"io"
	"mime/multipart"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/httputil"
	"github.com/diamondburned/arikawa/v2/utils/json"
)

const AttachmentSpoilerPrefix = "SPOILER_"

// AllowedMentions is a allowlist of mentions for a message.
//
// Allowlists
//
// Roles and Users are slices that act as allowlists for IDs that are allowed
// to be mentioned. For example, if only 1 ID is provided in Users, then only
// that ID will be parsed in the message. No other IDs will be. The same
// example also applies for roles.
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
// Roles and Users.
//
// https://discordapp.com/developers/docs/resources/channel#allowed-mentions-object
type AllowedMentions struct {
	// Parse is an array of allowed mention types to parse from the content.
	Parse []AllowedMentionType `json:"parse"`
	// Roles is an array of role_ids to mention (Max size of 100).
	Roles []discord.RoleID `json:"roles,omitempty"`
	// Users is an array of user_ids to mention (Max size of 100).
	Users []discord.UserID `json:"users,omitempty"`
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
		return errors.Errorf("roles slice length %d is over 100", len(am.Roles))
	}
	if len(am.Users) > 100 {
		return errors.Errorf("users slice length %d is over 100", len(am.Users))
	}

	for _, allowed := range am.Parse {
		switch allowed {
		case AllowRoleMention:
			if len(am.Roles) > 0 {
				return errors.New(`parse has AllowRoleMention and Roles slice is not empty`)
			}
		case AllowUserMention:
			if len(am.Users) > 0 {
				return errors.New(`parse has AllowUserMention and Users slice is not empty`)
			}
		}
	}

	return nil
}

// ErrEmptyMessage is returned if either a SendMessageData or an
// ExecuteWebhookData has both an empty Content and no Embed(s).
var ErrEmptyMessage = errors.New("message is empty")

// SendMessageFile represents a file to be uploaded to Discord.
type SendMessageFile struct {
	Name   string
	Reader io.Reader
}

// SendMessageData is the full structure to send a new message to Discord with.
type SendMessageData struct {
	// Content are the message contents (up to 2000 characters).
	Content string `json:"content,omitempty"`
	// Nonce is a nonce that can be used for optimistic message sending.
	Nonce string `json:"nonce,omitempty"`

	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
	// Embed is embedded rich content.
	Embed *discord.Embed `json:"embed,omitempty"`

	Files []SendMessageFile `json:"-"`

	// AllowedMentions are the allowed mentions for a message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
}

func (data *SendMessageData) WriteMultipart(body *multipart.Writer) error {
	return writeMultipart(body, data, data.Files)
}

// SendMessageComplex posts a message to a guild text or DM channel. If
// operating on a guild channel, this endpoint requires the SEND_MESSAGES
// permission to be present on the current user. If the tts field is set to
// true, the SEND_TTS_MESSAGES permission is required for the message to be
// spoken. Returns a message object. Fires a Message Create Gateway event.
//
// The maximum request size when sending a message is 8MB.
//
// This endpoint supports requests with Content-Types of both application/json
// and multipart/form-data. You must however use multipart/form-data when
// uploading files. Note that when sending multipart/form-data requests the
// embed field cannot be used, however you can pass a JSON encoded body as form
// value for payload_json, where additional request parameters such as embed
// can be set.
//
// Note that when sending application/json you must send at least one of
// content or embed, and when sending multipart/form-data, you must send at
// least one of content, embed or file. For a file attachment, the
// Content-Disposition subpart header MUST contain a filename parameter.
func (c *Client) SendMessageComplex(
	channelID discord.ChannelID, data SendMessageData) (*discord.Message, error) {

	if data.Content == "" && data.Embed == nil && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
	}

	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}

	if data.Embed != nil {
		if err := data.Embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "embed error")
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

	Files []SendMessageFile `json:"-"`

	// AllowedMentions are the allowed mentions for the message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
}

func (data *ExecuteWebhookData) WriteMultipart(body *multipart.Writer) error {
	return writeMultipart(body, data, data.Files)
}

func writeMultipart(body *multipart.Writer, item interface{}, files []SendMessageFile) error {
	defer body.Close()

	// Encode the JSON body first
	w, err := body.CreateFormField("payload_json")
	if err != nil {
		return errors.Wrap(err, "failed to create bodypart for JSON")
	}

	if err := json.EncodeStream(w, item); err != nil {
		return errors.Wrap(err, "failed to encode JSON")
	}

	for i, file := range files {
		num := strconv.Itoa(i)

		w, err := body.CreateFormFile("file"+num, file.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create bodypart for "+num)
		}

		if _, err := io.Copy(w, file.Reader); err != nil {
			return errors.Wrap(err, "failed to write for file "+num)
		}
	}

	return nil
}
