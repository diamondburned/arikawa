package api

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/internal/json"
	"github.com/pkg/errors"
)

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

type SendMessageFile struct {
	Name        string
	ContentType string // auto-detect if empty
	Reader      io.Reader
}

type SendMessageData struct {
	Content string `json:"content,omitempty"`
	Nonce   string `json:"nonce,omitempty"`
	TTS     bool   `json:"tts"`

	Embed *discord.Embed `json:"embed,omitempty"`

	Files []SendMessageFile `json:"-"`
}

func (data *SendMessageData) WriteMultipart(
	c json.Driver, w io.Writer) error {

	return writeMultipart(c, w, data, data.Files)
}

type ExecuteWebhookData struct {
	SendMessageData

	Username  string      `json:"username,omitempty"`
	AvatarURL discord.URL `json:"avatar_url,omitempty"`
}

func (data *ExecuteWebhookData) WriteMultipart(
	c json.Driver, w io.Writer) error {

	return writeMultipart(c, w, data, data.Files)
}

func writeMultipart(
	c json.Driver, w io.Writer,
	item interface{}, files []SendMessageFile) error {

	body := multipart.NewWriter(w)

	// Encode the JSON body first
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="payload_json"`)
	h.Set("Content-Type", "application/json")

	w, err := body.CreatePart(h)
	if err != nil {
		return errors.Wrap(err, "Failed to create bodypart for JSON")
	}

	j, err := c.Marshal(item)
	log.Println(string(j), err)

	if err := c.EncodeStream(w, item); err != nil {
		return errors.Wrap(err, "Failed to encode JSON")
	}

	// Content-Type buffer
	var buf []byte

	for i, file := range files {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(
			`form-data; name="file%d"; filename="%s"`,
			i, quoteEscaper.Replace(file.Name),
		))

		var bufUsed int

		if file.ContentType == "" {
			if buf == nil {
				buf = make([]byte, 512)
			}

			n, err := file.Reader.Read(buf)
			if err != nil {
				return errors.Wrap(err, "Failed to read first 512 bytes for "+
					strconv.Itoa(i))
			}

			file.ContentType = http.DetectContentType(buf[:n])
			files[i] = file
			bufUsed = n
		}

		h.Set("Content-Type", file.ContentType)

		w, err := body.CreatePart(h)
		if err != nil {
			return errors.Wrap(err, "Failed to create bodypart for "+
				strconv.Itoa(i))
		}

		if bufUsed > 0 {
			// Prematurely write
			if _, err := w.Write(buf[:bufUsed]); err != nil {
				return errors.Wrap(err, "Failed to write buffer for "+
					strconv.Itoa(i))
			}
		}

		if _, err := io.Copy(w, file.Reader); err != nil {
			return errors.Wrap(err, "Failed to write file for "+
				strconv.Itoa(i))
		}
	}

	if err := body.Close(); err != nil {
		return errors.Wrap(err, "Failed to close body writer")
	}

	return nil
}
