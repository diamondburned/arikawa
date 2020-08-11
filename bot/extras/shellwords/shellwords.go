package shellwords

import (
	"fmt"
	"strings"
)

type ErrParse struct {
	Position int
	ErrorStart,
	ErrorPart,
	ErrorEnd string
}

func (e ErrParse) Error() string {
	return fmt.Sprintf(
		"Unexpected quote or escape: %s__%s__%s",
		e.ErrorStart, e.ErrorPart, e.ErrorEnd,
	)
}

// Parse parses the given text to a slice of words.
func Parse(line string) ([]string, error) {
	var args []string
	var escaped, doubleQuoted, singleQuoted bool

	var buf strings.Builder
	buf.Grow(len(line))

	got := false
	cursor := 0

	runes := []rune(line)

	for _, r := range runes {
		if escaped {
			buf.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if singleQuoted {
				buf.WriteRune(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			switch {
			case singleQuoted, doubleQuoted:
				buf.WriteRune(r)
			case got:
				cursor += buf.Len()
				args = append(args, buf.String())
				buf.Reset()
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
		case '\'', '`':
			if !doubleQuoted {
				if singleQuoted {
					got = true
				}

				// // If this is a backtick, then write it.
				// if r == '`' {
				// 	buf.WriteByte('`')
				// }

				singleQuoted = !singleQuoted
				continue
			}
		}

		got = true
		buf.WriteRune(r)
	}

	if got {
		args = append(args, buf.String())
	}

	if escaped || singleQuoted || doubleQuoted {
		// the number of characters to highlight
		var (
			pos   = cursor + 5
			start = string(runes[max(cursor-100, 0) : pos-1])
			end   = string(runes[pos+1 : min(cursor+100, len(runes))])
			part  = string(runes[max(pos-1, 0):min(len(runes), pos+2)])
		)

		return args, &ErrParse{
			Position:   cursor,
			ErrorStart: start,
			ErrorPart:  part,
			ErrorEnd:   end,
		}
	}

	return args, nil
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n', '　':
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
