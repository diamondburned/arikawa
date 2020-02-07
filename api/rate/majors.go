package rate

import (
	"strconv"
	"strings"
)

// TODO: webhook
var MajorRootPaths = []string{"channels", "guilds"}

func ParseBucketKey(path string) string {
	path = strings.SplitN(path, "?", 2)[0]

	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		return path
	}

	parts = parts[1:] // [0] is just "" since URL

	var skip int

	for _, part := range MajorRootPaths {
		if part == parts[0] {
			skip = 2
			break
		}
	}

	// We add 1, since this is the string path. The path following this would be
	// the actual value, which we check.
	skip++

	// we need to remove IDs from these
	for ; skip < len(parts); skip += 2 {
		// Check if it's a number:
		if _, err := strconv.Atoi(parts[skip]); err == nil {
			parts[skip] = ""
			continue
		}

		// Check if it's an emoji:
		if StringIsEmojiOnly(parts[skip]) {
			parts[skip] = ""
			continue
		}

		// Check if it's a custom emoji:
		if StringIsCustomEmoji(parts[skip]) {
			parts[skip] = ""
			continue
		}
	}

	// rejoin url
	path = strings.Join(parts, "/")
	return "/" + path
}
