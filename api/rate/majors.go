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

	// we need to remove IDs from these
	for ; skip < len(parts); skip++ {
		if _, err := strconv.Atoi(parts[skip]); err == nil {
			// is a number, DELET
			parts[skip] = ""
		}
	}

	// rejoin url
	path = strings.Join(parts, "/")
	return "/" + path
}
