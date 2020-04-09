package api

import (
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/utils/httputil"
	"github.com/diamondburned/arikawa/utils/json"
	"github.com/pkg/errors"
)

func (c *Client) SendMessageComplex(
	channelID discord.Snowflake,
	data SendMessageData) (*discord.Message, error) {

	if data.Embed != nil {
		if err := data.Embed.Validate(); err != nil {
			return nil, errors.Wrap(err, "Embed error")
		}
	}

	var URL = EndpointChannels + channelID.String() + "/messages"
	var msg *discord.Message

	if len(data.Files) == 0 {
		// No files, so no need for streaming.
		return msg, c.RequestJSON(&msg, "POST", URL,
			httputil.WithJSONBody(c, data))
	}

	writer := func(mw *multipart.Writer) error {
		return data.WriteMultipart(c, mw)
	}

	resp, err := c.MeanwhileMultipart(writer, "POST", URL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return msg, c.DecodeStream(resp.Body, &msg)
}

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
