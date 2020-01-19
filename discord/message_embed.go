package discord

import "fmt"

type Color uint32

const DefaultColor Color = 0x303030

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

func NewEmbed() *Embed {
	return &Embed{
		Type:  NormalEmbed,
		Color: DefaultColor,
	}
}

type ErrOverbound struct {
	Count int
	Max   int

	Thing string
}

var _ error = (*ErrOverbound)(nil)

func (e ErrOverbound) Error() string {
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
		e.Color = DefaultColor
	}

	if len(e.Title) > 256 {
		return &ErrOverbound{len(e.Title), 256, "Title"}
	}

	if len(e.Description) > 2048 {
		return &ErrOverbound{len(e.Description), 2048, "Description"}
	}

	if len(e.Fields) > 25 {
		return &ErrOverbound{len(e.Fields), 25, "Fields"}
	}

	var sum = 0 +
		len(e.Title) +
		len(e.Description)

	if e.Footer != nil {
		if len(e.Footer.Text) > 2048 {
			return &ErrOverbound{len(e.Footer.Text), 2048, "Footer text"}
		}

		sum += len(e.Footer.Text)
	}

	if e.Author != nil {
		if len(e.Author.Name) > 256 {
			return &ErrOverbound{len(e.Author.Name), 256, "Author name"}
		}

		sum += len(e.Author.Name)
	}

	for i, field := range e.Fields {
		if len(field.Name) > 256 {
			return &ErrOverbound{len(field.Name), 256,
				fmt.Sprintf("field %d name", i)}
		}

		if len(field.Value) > 1024 {
			return &ErrOverbound{len(field.Value), 1024,
				fmt.Sprintf("field %d value", i)}
		}

		sum += len(field.Name) + len(field.Value)
	}

	if sum > 6000 {
		return &ErrOverbound{sum, 6000, "Sum of all characters"}
	}

	return nil
}

type EmbedType string

const (
	NormalEmbed EmbedType = "rich"
	ImageEmbed  EmbedType = "image"
	VideoEmbed  EmbedType = "video"
	// Undocumented
)

type EmbedFooter struct {
	Text      string `json:"text"`
	Icon      URL    `json:"icon_url,omitempty"`
	ProxyIcon URL    `json:"proxy_icon_url,omitempty"`
}

type EmbedImage struct {
	URL   URL `json:"url"`
	Proxy URL `json:"proxy_url"`
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
