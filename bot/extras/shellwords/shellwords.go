package shellwords

import (
	"fmt"
	"strings"
)

// WordOffset is the offset from the position cursor to print on the error.
const WordOffset = 7

var escaper = strings.NewReplacer(
	"`", "\\`",
	"@", "\\@",
	"\\", "\\\\",
)

type ErrParse struct {
	Position int
	Words    string // joined
}

func (e ErrParse) Error() string {
	// Magic number 5.
	var a = max(0, e.Position-WordOffset)
	var b = min(len(e.Words), e.Position+WordOffset)
	var word = e.Words[a:b]
	var uidx = e.Position - a

	errstr := strings.Builder{}
	errstr.WriteString("Unexpected quote or escape")

	// Do a bound check.
	if uidx+1 > len(word) {
		// Invalid.
		errstr.WriteString(".")
		return errstr.String()
	}

	// Write the pre-underline part.
	fmt.Fprintf(
		&errstr, ": %s__%s__",
		escaper.Replace(word[:uidx]),
		escaper.Replace(string(word[uidx:])),
	)

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
		return args, &ErrParse{
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
