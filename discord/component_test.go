package discord

import (
	"log"
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

var whereErrorV = []Component{
	ActionRowComponent{
		SelectComponent{
			CustomID: "blog.display",
			Options: []SelectOption{
				{
					Label:       "Code of Conduct Updates",
					Value:       "https://go.dev/blog/conduct-2021",
					Description: "Carmen Andoh, Russ Cox,  and Steve Francia",
					Default:     false,
				},
				{
					Label:       "Tidying up the Go web experience",
					Value:       "https://go.dev/blog/tidy-web",
					Description: "Russ Cox",
					Default:     false,
				},
				{
					Label:       "The Go Collective on Stack Overflow",
					Value:       "https://go.dev/blog/stackoverflow",
					Description: "Steve Francia",
					Default:     false,
				},
				{
					Label:       "Go Developer Survey 2020 Results",
					Value:       "https://go.dev/blog/survey2020-results",
					Description: "Alice Merrick",
					Default:     false,
				},
				{
					Label:       "Gopls on by default in the VS Code Go extension",
					Value:       "https://go.dev/blog/gopls-vscode-go",
					Description: "Go tools team",
					Default:     false,
				},
			},
			Placeholder: "Display Blog Post",
			Disabled:    false,
		},
	},
	ActionRowComponent{
		ActionRowComponent{
			ButtonComponent{
				Label:    "Prev Page",
				CustomID: "blog.prev.the",
				Style:    SecondaryButtonStyle(),
				Emoji: &ComponentEmoji{
					Name:     "⬅️",
					ID:       0x0000000000000000,
					Animated: false,
				},
				Disabled: false,
			},
			ButtonComponent{
				Label:    "Next Page",
				CustomID: "blog.next.the",
				Style:    SecondaryButtonStyle(),
				Emoji: &ComponentEmoji{
					Name:     "➡️",
					ID:       0x0000000000000000,
					Animated: false,
				},
				Disabled: false,
			},
		},
	},
}

const whereErrorMsg = `{
  "code": 50035,
  "errors": {
    "data": {
      "components": {
        "1": {
          "components": {
            "0": {
              "_errors": [
                {
                  "code": "COMPONENT_TYPE_INVALID",
                  "message": "The specified component type is invalid in this context"
                }
              ]
            }
          }
        }
      }
    }
  },
  "message": "Invalid Form Body"
}`

func TestWhereError(t *testing.T) {
	whereErrorMsg := httputil.HTTPError{
		Status:  400,
		Body:    []byte(whereErrorMsg),
		Code:    50035,
		Message: "Invalid Form Body",
	}

	errors := WhereComponentError(whereErrorMsg, whereErrorV)
	log.Printf("%#v", errors)
}
