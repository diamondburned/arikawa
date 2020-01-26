package shellwords

import (
	"errors"
)

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

type Parser struct {
	Position int
}

func NewParser() *Parser {
	return &Parser{
		Position: 0,
	}
}

func (p *Parser) Parse(line string) ([]string, error) {
	args := []string{}
	buf := ""
	var escaped, doubleQuoted, singleQuoted, backQuote bool
	backtick := ""

	pos := -1
	got := false

	for _, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if singleQuoted {
				buf += string(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			switch {
			case singleQuoted, doubleQuoted, backQuote:
				buf += string(r)
				backtick += string(r)
			case got:
				args = append(args, buf)
				buf = ""
				got = false
			}
			continue
		}

		switch r {
		case '`':
			if !singleQuoted && !doubleQuoted {
				backtick = ""
				backQuote = !backQuote
			}
		case '"':
			if !singleQuoted {
				if doubleQuoted {
					got = true
				}
				doubleQuoted = !doubleQuoted
				continue
			}
		case '\'':
			if !doubleQuoted {
				if singleQuoted {
					got = true
				}
				singleQuoted = !singleQuoted
				continue
			}
		}

		got = true
		buf += string(r)
		if backQuote {
			backtick += string(r)
		}
	}

	if got {
		args = append(args, buf)
	}

	if escaped || singleQuoted || doubleQuoted || backQuote {
		return nil, errors.New("invalid command line string")
	}

	p.Position = pos

	return args, nil
}

func Parse(line string) ([]string, error) {
	return NewParser().Parse(line)
}
