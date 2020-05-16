package discord

import "strings"

type ImageType string

const (
	// AutoImage chooses automatically between a PNG and GIF.
	AutoImage ImageType = "auto"

	// JPEGImage is the JPEG image type.
	JPEGImage ImageType = ".jpeg"
	// PNGImage is the PNG image type.
	PNGImage ImageType = ".png"
	// WebPImage is the WebP image type.
	WebPImage ImageType = ".webp"
	// GIFImage is the GIF image type.
	GIFImage ImageType = ".gif"
)

func (t ImageType) format(name string) string {
	if t == AutoImage {
		if strings.HasPrefix(name, "a_") {
			return name + ".gif"
		}

		return name + ".png"
	}

	return name + string(t)
}

type URL = string
type Hash = string
