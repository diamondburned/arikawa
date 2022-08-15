package discord_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

func ExampleContainerComponents_Unmarshal() {
	components := &discord.ContainerComponents{
		&discord.ActionRowComponent{
			&discord.TextInputComponent{
				CustomID: "text1",
				Value:    option.NewNullableString("hello"),
			},
		},
		&discord.ActionRowComponent{
			&discord.TextInputComponent{
				CustomID: "text2",
				Value:    option.NewNullableString("hello 2"),
			},
			&discord.TextInputComponent{
				CustomID: "text3",
				Value:    option.NewNullableString("hello 3"),
			},
		},
		&discord.ActionRowComponent{
			&discord.SelectComponent{
				CustomID: "select1",
				Options: []discord.SelectOption{
					{Value: "option 1"},
					{Value: "option 2"},
				},
			},
			&discord.ButtonComponent{
				CustomID: "button1",
			},
		},
		&discord.ActionRowComponent{
			&discord.SelectComponent{
				CustomID: "select2",
				Options: []discord.SelectOption{
					{Value: "option 1"},
				},
			},
		},
	}

	var data struct {
		Text1   string   `discord:"text1"`
		Text2   string   `discord:"text2?"`
		Text3   *string  `discord:"text3"`
		Text4   string   `discord:"text4?"`
		Text5   *string  `discord:"text5"`
		Select1 []string `discord:"select1"`
		Select2 string   `discord:"select2"`
		Button1 bool     `discord:"button1"`
	}

	if err := components.Unmarshal(&data); err != nil {
		log.Fatalln(err)
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))

	// Output:
	// {
	//   "Text1": "hello",
	//   "Text2": "hello 2",
	//   "Text3": "hello 3",
	//   "Text4": "",
	//   "Text5": null,
	//   "Select1": [
	//     "option 1",
	//     "option 2"
	//   ],
	//   "Select2": "option 1",
	//   "Button1": true
	// }
}
