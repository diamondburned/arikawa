package api

import (
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/internal/json"
	"github.com/pkg/errors"
)

const AttachmentSpoilerPrefix = "SPOILER_"

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

type SendMessageFile struct {
	Name   string
	Reader io.Reader
}

type SendMessageData struct {
	Content string `json:"content,omitempty"`
	Nonce   string `json:"nonce,omitempty"`
	TTS     bool   `json:"tts"`

	Embed *discord.Embed `json:"embed,omitempty"`

	Files []SendMessageFile `json:"-"`
}

func (data *SendMessageData) WriteMultipart(
	c json.Driver, body *multipart.Writer) error {

	return writeMultipart(c, body, data, data.Files)
}

type ExecuteWebhookData struct {
	Content string `json:"content,omitempty"`
	Nonce   string `json:"nonce,omitempty"`
	TTS     bool   `json:"tts"`

	Embeds []discord.Embed `json:"embeds,omitempty"`

	Files []SendMessageFile `json:"-"`

	Username  string      `json:"username,omitempty"`
	AvatarURL discord.URL `json:"avatar_url,omitempty"`
}

func (data *ExecuteWebhookData) WriteMultipart(
	c json.Driver, body *multipart.Writer) error {

	return writeMultipart(c, body, data, data.Files)
}

func writeMultipart(
	c json.Driver, body *multipart.Writer,
	item interface{}, files []SendMessageFile) error {

	defer body.Close()

	// Encode the JSON body first
	w, err := body.CreateFormField("payload_json")
	if err != nil {
		return errors.Wrap(err, "Failed to create bodypart for JSON")
	}

	if err := c.EncodeStream(w, item); err != nil {
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
