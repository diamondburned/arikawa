package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

var ErrInvalidImageCT = errors.New("unknown image content-type")
var ErrInvalidImageData = errors.New("invalid image data")

type ImageTooLargeError struct {
	Size, Max int
}

func (err ImageTooLargeError) Error() string {
	return fmt.Sprintf("Image is %.02fkb, larger than %.02fkb",
		float64(err.Size)/1000, float64(err.Max)/1000)
}

// Image wraps around the Data URI Scheme that Discord uses:
// https://discord.com/developers/docs/reference#image-data
type Image struct {
	// ContentType is optional and will be automatically detected. However, it
	// should always return "image/jpeg," "image/png" or "image/gif".
	ContentType string

	// Just raw content of the file.
	Content []byte
}

func DecodeImage(data []byte) (*Image, error) {
	parts := bytes.SplitN(data, []byte{';'}, 2)
	if len(parts) < 2 {
		return nil, ErrInvalidImageData
	}

	if !bytes.HasPrefix(parts[0], []byte("data:")) {
		return nil, errors.Wrap(ErrInvalidImageData, "invalid header")
	}

	if !bytes.HasPrefix(parts[1], []byte("base64,")) {
		return nil, errors.Wrap(ErrInvalidImageData, "invalid base64")
	}

	var b64 = parts[1][len("base64,"):]
	var img = Image{
		ContentType: string(parts[0][len("data:"):]),
		Content:     make([]byte, base64.StdEncoding.DecodedLen(len(b64))),
	}

	base64.StdEncoding.Decode(img.Content, b64)
	return &img, nil
}

func (i Image) Validate(maxSize int) error {
	if maxSize > 0 && len(i.Content) > maxSize {
		return ImageTooLargeError{len(i.Content), maxSize}
	}

	switch i.ContentType {
	case "image/png", "image/jpeg", "image/gif":
		return nil
	default:
		return ErrInvalidImageCT
	}
}

func (i Image) Encode() ([]byte, error) {
	if i.ContentType == "" {
		var max = 512
		if len(i.Content) < max {
			max = len(i.Content)
		}
		i.ContentType = http.DetectContentType(i.Content[:max])
	}

	if err := i.Validate(0); err != nil {
		return nil, err
	}

	b64enc := make([]byte, base64.StdEncoding.EncodedLen(len(i.Content)))
	base64.StdEncoding.Encode(b64enc, i.Content)

	return bytes.Join([][]byte{
		[]byte("data:"),
		[]byte(i.ContentType),
		[]byte(";base64,"),
		b64enc,
	}, nil), nil
}

var _ json.Marshaler = (*Image)(nil)
var _ json.Unmarshaler = (*Image)(nil)

func (i Image) MarshalJSON() ([]byte, error) {
	if len(i.Content) == 0 {
		return []byte("null"), nil
	}

	b, err := i.Encode()
	if err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{{'"'}, b, {'"'}}, nil), nil
}

func (i *Image) UnmarshalJSON(v []byte) error {
	// Trim string
	v = bytes.Trim(v, `"`)

	// Accept a nil image.
	if string(v) == "null" {
		return nil
	}

	img, err := DecodeImage(v)
	if err != nil {
		return err
	}

	*i = *img
	return nil
}
