package discord_test

import (
	"encoding/json"
	"fmt"
	"log"

	"libdb.so/arikawa/v4/discord"
	internaljson "libdb.so/arikawa/v4/utils/json"
)

func ExampleCommandInteractionOptions_Unmarshal() {
	options := discord.CommandInteractionOptions{
		opt(discord.ChannelOptionType, "channel_id", 1),
		opt(discord.StringOptionType, "string1", "hello"),
		opt(discord.StringOptionType, "string2", "hello"),
		opt(discord.StringOptionType, "string3", "hello"),
		opt(discord.SubcommandOptionType, "sub", discord.CommandInteractionOptions{
			{
				Type:  discord.RoleOptionType,
				Name:  "role_id",
				Value: mustJSON("2"),
			},
		}),
	}

	var quickCommand struct {
		ChannelID       discord.ChannelID `discord:"channel_id"`
		String1         string
		OptionalString2 *string `discord:"string2"`
		OptionalString3 string  `discord:"string3?"`
		OptionalString4 *string `discord:"string4"`
		OptionalString5 string  `discord:"string5?"`
		Suboption       struct {
			RoleID discord.RoleID `discord:"role_id"`
		} `discord:"sub"`
		OptionalSuboption2 *struct {
			RoleID discord.RoleID `discord:"role_id"`
		} `discord:"sub2"`
		OptionalSuboption3 struct {
			RoleID discord.RoleID `discord:"role_id"`
		} `discord:"sub3?"`
	}

	if err := options.Unmarshal(&quickCommand); err != nil {
		log.Fatalln(err)
	}

	b, _ := json.MarshalIndent(quickCommand, "", "  ")
	fmt.Println(string(b))

	// Output:
	// {
	//   "ChannelID": "1",
	//   "String1": "hello",
	//   "OptionalString2": "hello",
	//   "OptionalString3": "hello",
	//   "OptionalString4": null,
	//   "OptionalString5": "",
	//   "Suboption": {
	//     "RoleID": "2"
	//   },
	//   "OptionalSuboption2": null,
	//   "OptionalSuboption3": {
	//     "RoleID": null
	//   }
	// }
}

func opt(t discord.CommandOptionType, name string, v interface{}) discord.CommandInteractionOption {
	o := discord.CommandInteractionOption{
		Type: t,
		Name: name,
	}
	if opts, ok := v.(discord.CommandInteractionOptions); ok {
		o.Options = opts
	} else {
		o.Value = mustJSON(v)
	}
	return o
}

func mustJSON(v interface{}) internaljson.Raw {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return internaljson.Raw(b)
}
