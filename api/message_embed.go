package api

import (
	"fmt"

	"git.sr.ht/~diamondburned/arikawa/discord"
)

type Embed struct {
	Title       string    `json:"title,omitempty"`
	Type        EmbedType `json:"type,omitempty"`
	Description string    `json:"description,omitempty"`

	URL discord.URL `json:"url,omitempty"`

	Timestamp discord.Timestamp `json:"timestamp,omitempty"`
	Color     discord.Color     `json:"color,omitempty"`

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
		Color: discord.DefaultColor,
	}
}

func (e *Embed) Validate() error {
	if e.Type == "" {
		e.Type = NormalEmbed
	}

	if e.Color == 0 {
		e.Color = discord.DefaultColor
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
				fmt.Sprintf("field %s value", i)}
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
	NormalEmbed = "rich"
	ImageEmbed  = "image"
	VideoEmbed  = "video"
	// Undocumented
)

type EmbedFooter struct {
	Text      string      `json:"text"`
	Icon      discord.URL `json:"icon_url,omitempty"`
	ProxyIcon discord.URL `json:"proxy_icon_url,omitempty"`
}

type EmbedImage struct {
	URL   discord.URL `json:"url"`
	Proxy discord.URL `json:"proxy_url"`
}

type EmbedThumbnail struct {
	URL    discord.URL `json:"url,omitempty"`
	Proxy  discord.URL `json:"proxy_url,omitempty"`
	Height uint        `json:"height,omitempty"`
	Width  uint        `json:"width,omitempty"`
}

type EmbedVideo struct {
	URL    discord.URL `json:"url"`
	Height uint        `json:"height"`
	Width  uint        `json:"width"`
}

type EmbedProvider struct {
	Name string      `json:"name"`
	URL  discord.URL `json:"url"`
}

type EmbedAuthor struct {
	Name      string      `json:"name,omitempty"`
	URL       discord.URL `json:"url,omitempty"`
	Icon      discord.URL `json:"icon_url,omitempty"`
	ProxyIcon discord.URL `json:"proxy_icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}
