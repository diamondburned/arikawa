package arguments

import (
	"errors"
	"regexp"
)

var (
	ChannelRegex = regexp.MustCompile(`<#(\d+)>`)
	UserRegex    = regexp.MustCompile(`<@!?(\d+)>`)
	RoleRegex    = regexp.MustCompile(`<@&(\d+)>`)
)

type ChannelMention string

func (m *ChannelMention) Parse(arg string) error {
	return grabFirst(ChannelRegex, "channel mention", arg, (*string)(m))
}

func (m *ChannelMention) Usage() string {
	return "#channel"
}

type UserMention string

func (m *UserMention) Parse(arg string) error {
	return grabFirst(UserRegex, "user mention", arg, (*string)(m))
}

func (m *UserMention) Usage() string {
	return "@user"
}

type RoleMention string

func (m *RoleMention) Parse(arg string) error {
	return grabFirst(RoleRegex, "role mention", arg, (*string)(m))
}

func (m *RoleMention) Usage() string {
	return "@role"
}

func grabFirst(reg *regexp.Regexp, item, input string, output *string) error {
	matches := reg.FindStringSubmatch(input)
	if len(matches) < 2 {
		return errors.New("Invalid " + item)
	}

	*output = matches[1]
	return nil
}
