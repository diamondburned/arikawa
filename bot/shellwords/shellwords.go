package shellwords

import (
	"errors"
)

type ErrParse struct {
	Line     string
	Position string
}

func Parse(line string) ([]string, error) {
	args := []string{}
	buf := ""
	var escaped, doubleQuoted, singleQuoted bool
	backtick := ""

	got := false
	cursor := 0

	runes := []rune(line)

	for _, r := range runes {
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
			case singleQuoted, doubleQuoted:
				buf += string(r)
				backtick += string(r)
			case got:
				cursor += len(buf)
				args = append(args, buf)
				buf = ""
				got = false
			}
			continue
		}

		switch r {
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
	}

	if got {
		args = append(args, buf)
	}

	if escaped || singleQuoted || doubleQuoted {
		// the number of characters to highlight
		var (
			pos   = cursor + 5
			start = string(runes[max(cursor-100, 0) : pos-1])
			end   = string(runes[pos+1 : min(cursor+100, len(runes))])
			part  = ""
		)

		for i := pos - 1; i >= 0 && i < len(runes) && i < pos+2; i++ {
			if runes[i] == '\\' {
				part += "\\"
			}
			part += string(runes[i])
		}

		return nil, errors.New(
			"Unexpected quote or escape: " + start + "__" + part + "__" + end)
	}

	return args, nil
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func max(i, j int) int {
	if i < j {
		return j
	}
	return i
}
