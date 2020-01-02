package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"git.sr.ht/~diamondburned/arikawa/discord"
	"git.sr.ht/~diamondburned/arikawa/json"
	"github.com/pkg/errors"
)

type SendMessageData struct {
	Content string `json:"content"`
	Nonce   string `json:"nonce"`
	TTS     bool   `json:"tts"`

	Embed *discord.Embed `json:"embed"`

	Files []SendMessageFile `json:"-"`
}

type SendMessageFile struct {
	Name        string
	ContentType string // auto-detect if empty
	Reader      io.Reader
}

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func (data *SendMessageData) WriteMultipart(c json.Driver, w io.Writer) error {
	body := multipart.NewWriter(w)

	// Encode the JSON body first
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="payload_json"`)
	h.Set("Content-Type", "application/json")

	w, err := body.CreatePart(h)
	if err != nil {
		return errors.Wrap(err, "Failed to create bodypart for JSON")
	}

	if err := c.EncodeStream(w, data); err != nil {
		return errors.Wrap(err, "Failed to encode JSON")
	}

	// Content-Type buffer
	var buf []byte

	for i, file := range data.Files {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(
			`form-data; name="file%d"; filename="%s"`,
			i, quoteEscaper.Replace(file.Name),
		))

		w, err := body.CreatePart(h)
		if err != nil {
			return errors.Wrap(err, "Failed to create bodypart for "+
				strconv.Itoa(i))
		}

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
			data.Files[i] = file

			h.Set("Content-Type", file.ContentType)

			// Prematurely write
			if _, err := w.Write(buf[:n]); err != nil {
				return errors.Wrap(err, "Failed to write buffer for "+
					strconv.Itoa(i))
			}

		} else {
			h.Set("Content-Type", file.ContentType)
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
