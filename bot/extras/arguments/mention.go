package arguments

import (
	"errors"
	"regexp"

	"github.com/diamondburned/arikawa/v3/discord"
)

var (
	ChannelRegex = regexp.MustCompile(`<#(\d+)>`)
	UserRegex    = regexp.MustCompile(`<@!?(\d+)>`)
	RoleRegex    = regexp.MustCompile(`<@&(\d+)>`)
)

//

type ChannelMention discord.ChannelID

func (m *ChannelMention) Parse(arg string) error {
	return grabFirst(ChannelRegex, "channel mention", arg, (*discord.Snowflake)(m))
}

func (m *ChannelMention) Usage() string {
	return "#channel"
}

func (m *ChannelMention) ID() discord.ChannelID {
	return discord.ChannelID(*m)
}

func (m *ChannelMention) Mention() string {
	return "<#" + m.ID().String() + ">"
}

//

type UserMention discord.UserID

func (m *UserMention) Parse(arg string) error {
	return grabFirst(UserRegex, "user mention", arg, (*discord.Snowflake)(m))
}

func (m *UserMention) Usage() string {
	return "@user"
}

func (m *UserMention) ID() discord.UserID {
	return discord.UserID(*m)
}

func (m *UserMention) Mention() string {
	return "<@" + m.ID().String() + ">"
}

//

type RoleMention discord.RoleID

func (m *RoleMention) Parse(arg string) error {
	return grabFirst(RoleRegex, "role mention", arg, (*discord.Snowflake)(m))
}

func (m *RoleMention) Usage() string {
	return "@role"
}

func (m *RoleMention) ID() discord.RoleID {
	return discord.RoleID(*m)
}

func (m *RoleMention) Mention() string {
	return "<@&" + m.ID().String() + ">"
}

//

func grabFirst(reg *regexp.Regexp, item, input string, output *discord.Snowflake) error {
	matches := reg.FindStringSubmatch(input)
	if len(matches) < 2 {
		return errors.New("invalid " + item)
	}

	id, err := discord.ParseSnowflake(matches[1])
	if err != nil {
		return errors.New("invalid " + item)
	}

	*output = id
	return nil
}
