package arguments

import (
	"errors"
	"regexp"

	"github.com/diamondburned/arikawa/discord"
)

// (empty) so it matches standard links
// | OR
// canary. matches canary MessageURL
// 3 `(\d+)` for guild ID, channel ID and message ID
var Regex = regexp.MustCompile(
	`https://(ptb\.|canary\.)?discord(?:app)?\.com/channels/(\d+)/(\d+)/(\d+)`,
)

// MessageURL contains info from a MessageURL
type MessageURL struct {
	GuildID   discord.Snowflake
	ChannelID discord.Snowflake
	MessageID discord.Snowflake
}

func (url *MessageURL) Parse(arg string) error {
	u := ParseMessageURL(arg)
	if u == nil {
		return errors.New("Invalid MessageURL format.")
	}
	*url = *u
	return nil
}

func (url *MessageURL) Usage() string {
	return "https\u200b://discordapp.com/channels/\\*/\\*/\\*"
}

// ParseMessageURL parses the MessageURL into a smartlink
func ParseMessageURL(url string) *MessageURL {
	ss := Regex.FindAllStringSubmatch(url, -1)
	if ss == nil {
		return nil
	}

	if len(ss) == 0 || len(ss[0]) != 5 {
		return nil
	}

	gID, err1 := discord.ParseSnowflake(ss[0][2])
	cID, err2 := discord.ParseSnowflake(ss[0][3])
	mID, err3 := discord.ParseSnowflake(ss[0][4])

	if err1 != nil || err2 != nil || err3 != nil {
		return nil
	}

	return &MessageURL{
		GuildID:   gID,
		ChannelID: cID,
		MessageID: mID,
	}
}
