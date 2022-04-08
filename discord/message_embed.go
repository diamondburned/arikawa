package discord

import (
	"fmt"
	"strconv"
)

// Color describes an RGB color (with NO alpha). If a value is -1, then it's
// marshaled to JSON as null.
type Color int32

// DefaultEmbedColor is the default color to use for an embed.
var DefaultEmbedColor Color = 0x303030

// NullColor is a Color that's marshaled to null.
const NullColor Color = -1

// Uint32 returns the color as a Uint32. If the color is null, then 0 is
// returned.
func (c Color) Uint32() uint32 {
	if c == NullColor {
		return 0
	}
	return uint32(c)
}

// Int converts Color to int.
func (c Color) Int() int {
	return int(c)
}

// RGB splits Color into red, green, and blue. The maximum value is 255.
func (c Color) RGB() (uint8, uint8, uint8) {
	var (
		color = c.Uint32()

		r = uint8((color >> 16) & 255)
		g = uint8((color >> 8) & 255)
		b = uint8(color & 255)
	)

	return r, g, b
}

// String returns the Color in hexadecimal (#FFFFFF) format.
func (c Color) String() string {
	r, g, b := c.RGB()
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func (c Color) MarshalJSON() ([]byte, error) {
	if c < 0 {
		return []byte("null"), nil
	}
	return []byte(strconv.Itoa(c.Int())), nil
}

func (c *Color) UnmarshalJSON(json []byte) error {
	s := string(json)

	if s == "null" {
		*c = NullColor
		return nil
	}

	v, err := strconv.ParseInt(s, 10, 32)
	*c = Color(v)
	return err
}

// Embed describes a box with a left colored border that sometimes appears in
// messages.
type Embed struct {
	Title       string    `json:"title,omitempty"`
	Type        EmbedType `json:"type,omitempty"`
	Description string    `json:"description,omitempty"`

	URL URL `json:"url,omitempty"`

	Timestamp Timestamp `json:"timestamp,omitempty"`
	Color     Color     `json:"color,omitempty"`

	Footer    *EmbedFooter    `json:"footer,omitempty"`
	Image     *EmbedImage     `json:"image,omitempty"`
	Thumbnail *EmbedThumbnail `json:"thumbnail,omitempty"`
	Video     *EmbedVideo     `json:"video,omitempty"`
	Provider  *EmbedProvider  `json:"provider,omitempty"`
	Author    *EmbedAuthor    `json:"author,omitempty"`
	Fields    []EmbedField    `json:"fields,omitempty"`
}

// NewEmbed creates a normal embed with default values.
func NewEmbed() *Embed {
	return &Embed{
		Type:  NormalEmbed,
		Color: DefaultEmbedColor,
	}
}

// Validate validates the embed.
func (e *Embed) Validate() error {
	if e.Type == "" {
		e.Type = NormalEmbed
	}

	if e.Color == 0 {
		e.Color = DefaultEmbedColor
	}

	if len(e.Title) > 256 {
		return &OverboundError{len(e.Title), 256, "title"}
	}

	if len(e.Description) > 4096 {
		return &OverboundError{len(e.Description), 4096, "description"}
	}

	if len(e.Fields) > 25 {
		return &OverboundError{len(e.Fields), 25, "fields"}
	}

	if e.Footer != nil {
		if len(e.Footer.Text) > 2048 {
			return &OverboundError{len(e.Footer.Text), 2048, "footer text"}
		}
	}

	if e.Author != nil {
		if len(e.Author.Name) > 256 {
			return &OverboundError{len(e.Author.Name), 256, "author name"}
		}
	}

	for i, field := range e.Fields {
		if len(field.Name) > 256 {
			return &OverboundError{len(field.Name), 256,
				fmt.Sprintf("field %d name", i)}
		}

		if len(field.Value) > 1024 {
			return &OverboundError{len(field.Value), 1024,
				fmt.Sprintf("field %d value", i)}
		}
	}

	if sum := e.Length(); sum > 6000 {
		return &OverboundError{sum, 6000, "sum of all characters"}
	}

	return nil
}

// Length returns the sum of the lengths of all text in the embed.
func (e Embed) Length() int {
	var sum = 0 +
		len(e.Title) +
		len(e.Description)
	if e.Footer != nil {
		sum += len(e.Footer.Text)
	}
	if e.Author != nil {
		sum += len(e.Author.Name)
	}
	for _, field := range e.Fields {
		sum += len(field.Name) + len(field.Value)
	}
	return sum
}

// EmbedTypes are "loosely defined" and, for the most part, are not used by our
// clients for rendering. Embed attributes power what is rendered.
//
// Deprecated: Embed types should be considered deprecated and might be removed
// in a future API version.
type EmbedType string

// Embed type constants.
const (
	NormalEmbed  EmbedType = "rich"
	ImageEmbed   EmbedType = "image"
	VideoEmbed   EmbedType = "video"
	GIFVEmbed    EmbedType = "gifv"
	ArticleEmbed EmbedType = "article"
	LinkEmbed    EmbedType = "link"
)

// EmbedFooter is the footer of an embed.
type EmbedFooter struct {
	Text      string `json:"text"`
	Icon      URL    `json:"icon_url,omitempty"`
	ProxyIcon URL    `json:"proxy_icon_url,omitempty"`
}

// EmbedImage is the large image of an embed.
type EmbedImage struct {
	URL    URL  `json:"url"`
	Proxy  URL  `json:"proxy_url"`
	Height uint `json:"height,omitempty"`
	Width  uint `json:"width,omitempty"`
}

// EmbedThumbnail is the small image of an embed. It often appears on the right.
type EmbedThumbnail struct {
	URL    URL  `json:"url,omitempty"`
	Proxy  URL  `json:"proxy_url,omitempty"`
	Height uint `json:"height,omitempty"`
	Width  uint `json:"width,omitempty"`
}

// EmbedVideo is the video of an embed.
type EmbedVideo struct {
	URL    URL  `json:"url"`
	Proxy  URL  `json:"proxy_url,omitempty"`
	Height uint `json:"height"`
	Width  uint `json:"width"`
}

type EmbedProvider struct {
	Name string `json:"name"`
	URL  URL    `json:"url"`
}

type EmbedAuthor struct {
	Name      string `json:"name,omitempty"`
	URL       URL    `json:"url,omitempty"`
	Icon      URL    `json:"icon_url,omitempty"`
	ProxyIcon URL    `json:"proxy_icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}
