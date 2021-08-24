package api

import (
	"mime/multipart"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
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
// https://discord.com/developers/docs/resources/channel#allowed-mentions-object
type AllowedMentions struct {
	// Parse is an array of allowed mention types to parse from the content.
	Parse []AllowedMentionType `json:"parse"`
	// Roles is an array of role_ids to mention (Max size of 100).
	Roles []discord.RoleID `json:"roles,omitempty"`
	// Users is an array of user_ids to mention (Max size of 100).
	Users []discord.UserID `json:"users,omitempty"`
	// RepliedUser is used specifically for inline replies to specify, whether
	// to mention the author of the message you are replying to or not.
	RepliedUser option.Bool `json:"replied_user,omitempty"`
}

// AllowedMentionType is a constant that tells Discord what is allowed to parse
// from a message content. This can help prevent things such as an
// unintentional @everyone mention.
type AllowedMentionType string

// https://discord.com/developers/docs/resources/channel#allowed-mentions-object-allowed-mention-types
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
// ExecuteWebhookData is missing content, embeds, and files.
var ErrEmptyMessage = errors.New("message is empty")

// SendMessageData is the full structure to send a new message to Discord with.
type SendMessageData struct {
	// Content are the message contents (up to 2000 characters).
	Content string `json:"content,omitempty"`
	// Nonce is a nonce that can be used for optimistic message sending.
	Nonce string `json:"nonce,omitempty"`

	// TTS is true if this is a TTS message.
	TTS bool `json:"tts,omitempty"`
	// Embed is embedded rich content.
	Embeds []discord.Embed `json:"embeds,omitempty"`

	// Files is the list of file attachments to be uploaded. To reference a file
	// in an embed, use (sendpart.File).AttachmentURI().
	Files []sendpart.File `json:"-"`
	// Components is the list of components (such as buttons) to be attached to
	// the message.
	Components []discord.Component `json:"components,omitempty"`

	// AllowedMentions are the allowed mentions for a message.
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	// Reference allows you to reference another message to create a reply. The
	// referenced message must be from the same channel.
	//
	// Only MessageID is necessary. You may also include a channel_id and
	// guild_id in the reference. However, they are not necessary, but will be
	// validated if sent.
	Reference *discord.MessageReference `json:"message_reference,omitempty"`
}

// NeedsMultipart returns true if the SendMessageData has files.
func (data SendMessageData) NeedsMultipart() bool {
	return len(data.Files) > 0
}

func (data SendMessageData) WriteMultipart(body *multipart.Writer) error {
	return sendpart.Write(body, data, data.Files)
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
	if data.Content == "" && len(data.Embeds) == 0 && len(data.Files) == 0 {
		return nil, ErrEmptyMessage
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
			return nil, &discord.OverboundError{Count: sum, Max: 6000, Thing: "sum of all text in embeds"}
		}

		data.Embeds[i] = embed // embed.Validate changes fields
	}

	var URL = EndpointChannels + channelID.String() + "/messages"
	var msg *discord.Message
	return msg, sendpart.POST(c.Client, data, &msg, URL)
}
