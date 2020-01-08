package httputil

import "fmt"

// URL extends the normal URL and allows for a general string.
type URL struct {
	Base string
	URL  string
}

func URLf(base string, v ...interface{}) URL {
	return URL{base, fmt.Sprintf(base, v...)}
}

func (url URL) String() string { return url.URL }
