package shellwords

import (
	"strings"
)

var escaper = strings.NewReplacer(
	"__", "\\_\\_",
	"\\", "\\\\",
)

// ErrMissingClose is returned when the parsed line is missing a closing quote.
type ErrMissingClose struct {
	Position int
	Words    string // joined
}

func (e ErrMissingClose) Error() string {
	// Underline 7 characters around.
	var start = e.Position

	errstr := strings.Builder{}
	errstr.WriteString("missing quote close")

	if e.Words[start:] != "" {
		errstr.WriteString(": ")
		errstr.WriteString(escaper.Replace(e.Words[:start]))
		errstr.WriteString("__")
		errstr.WriteString(escaper.Replace(e.Words[start:]))
		errstr.WriteString("__")
	}

	return errstr.String()
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
		case '"', '“', '”':
			if !singleQuoted {
				if doubleQuoted {
					got = true
				}
				doubleQuoted = !doubleQuoted
				continue
			}
		case '\'', '`', '‘', '’':
			if !doubleQuoted {
				if singleQuoted {
					got = true
				}

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
		return args, ErrMissingClose{
			Position: cursor + buf.Len(),
			Words:    strings.Join(args, " "),
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
