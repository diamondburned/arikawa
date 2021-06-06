package discord

import "fmt"

type Color uint32

var DefaultEmbedColor Color = 0x303030

func (c Color) Uint32() uint32 {
	return uint32(c)
}

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

type Embed struct {
	Timestamp   Timestamp       `json:"timestamp,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Title       string          `json:"title,omitempty"`
	URL         URL             `json:"url,omitempty"`
	Description string          `json:"description,omitempty"`
	Type        EmbedType       `json:"type,omitempty"`
	Fields      []EmbedField    `json:"fields,omitempty"`
	Color       Color           `json:"color,omitempty"`
}

func NewEmbed() *Embed {
	return &Embed{
		Type:  NormalEmbed,
		Color: DefaultEmbedColor,
	}
}

type OverboundError struct {
  Thing string
	Count int
	Max   int
}

var _ error = (*OverboundError)(nil)

func (e OverboundError) Error() string {
	if e.Thing == "" {
		return fmt.Sprintf("Overbound error: %d > %d", e.Count, e.Max)
	}

	return fmt.Sprintf(e.Thing+" overbound: %d > %d", e.Count, e.Max)
}

func (e *Embed) Validate() error {
	if e.Type == "" {
		e.Type = NormalEmbed
	}

	if e.Color == 0 {
		e.Color = DefaultEmbedColor
	}

	if len(e.Title) > 256 {
		return &ErrOverbound{"title", len(e.Title), 256}
	}

	if len(e.Description) > 2048 {
		return &ErrOverbound{"description", len(e.Description), 2048}
	}

	if len(e.Fields) > 25 {
		return &ErrOverbound{"fields", len(e.Fields), 25}
	}

	sum := 0 +
		len(e.Title) +
		len(e.Description)

	if e.Footer != nil {
		if len(e.Footer.Text) > 2048 {
			return &ErrOverbound{"footer text", len(e.Footer.Text), 2048}
		}

		sum += len(e.Footer.Text)
	}

	if e.Author != nil {
		if len(e.Author.Name) > 256 {
			return &ErrOverbound{"author name", len(e.Author.Name), 256}
		}

		sum += len(e.Author.Name)
	}

	for i, field := range e.Fields {
		if len(field.Name) > 256 {
			return &ErrOverbound{fmt.Sprintf("field %d value", i), len(field.Name), 256}
		}

		if len(field.Value) > 1024 {
			return &ErrOverbound{fmt.Sprintf("field %d value", i), len(field.Value), 1024}

		}
		sum += len(field.Name) + len(field.Value)
	}

	if sum > 6000 {
		return &ErrOverbound{"sum of all characters", sum, 6000}
	}

	return nil
}

type EmbedType string

const (
	NormalEmbed  EmbedType = "rich"
	ImageEmbed   EmbedType = "image"
	VideoEmbed   EmbedType = "video"
	GIFVEmbed    EmbedType = "gifv"
	ArticleEmbed EmbedType = "article"
	LinkEmbed    EmbedType = "link"
	// Undocumented
)

type EmbedFooter struct {
	Text      string `json:"text"`
	Icon      URL    `json:"icon_url,omitempty"`
	ProxyIcon URL    `json:"proxy_icon_url,omitempty"`
}

type EmbedImage struct {
	URL    URL  `json:"url"`
	Proxy  URL  `json:"proxy_url"`
	Height uint `json:"height,omitempty"`
	Width  uint `json:"width,omitempty"`
}

type EmbedThumbnail struct {
	URL    URL  `json:"url,omitempty"`
	Proxy  URL  `json:"proxy_url,omitempty"`
	Height uint `json:"height,omitempty"`
	Width  uint `json:"width,omitempty"`
}

type EmbedVideo struct {
	URL    URL  `json:"url"`
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
