package sendpart

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json"
)

// File represents a file to be uploaded to Discord.
type File struct {
	Name   string
	Reader io.Reader
}

// AttachmentURI returns the file encoded using the attachment URI required for
// embedding an attachment image.
func (f File) AttachmentURI() string {
	u := url.URL{
		Scheme: "attachment",
		Path:   f.Name,
	}
	return u.String()
}

// GuessSize tries to guess the size of the [io.Reader] by checking if it
// implements [io.Seeker]. If not, then (0, false) is returned. This is useful
// if the reader is an [*os.File].
func (f File) GuessSize() (int64, bool) {
	seeker, ok := f.Reader.(io.Seeker)
	if !ok {
		return 0, false
	}

	// get current position. this makes no change to the file cursor.
	n1, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, false
	}

	// seek to the end of the file.
	n2, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		seeker.Seek(n1, io.SeekStart) // try to restore the cursor
		return 0, false
	}

	_, err = seeker.Seek(n1, io.SeekStart) // restore the cursor
	return n2 - n1, err == nil
}

// DataMultipartWriter is a MultipartWriter that also contains data that's
// JSON-marshalable.
type DataMultipartWriter interface {
	// NeedsMultipart returns true if the data interface must be sent using
	// multipart form.
	NeedsMultipart() bool

	httputil.MultipartWriter
}

// Do sends an HTTP request using client to the given URL and unmarshals the
// body into v if it's not nil. It will only send using multipart if needed.
func Do(c *httputil.Client, method string, data DataMultipartWriter, v any, url string) error {
	if !data.NeedsMultipart() {
		// No files, so no need for streaming.
		return c.RequestJSON(v, method, url, httputil.WithJSONBody(data))
	}

	resp, err := c.MeanwhileMultipart(data, method, url)
	if err != nil {
		return err
	}

	var body = resp.GetBody()
	defer body.Close()

	if v == nil {
		return nil
	}

	return json.DecodeStream(body, v)
}

// PATCH sends a PATCH request using client to the given URL and unmarshals the
// body into v if it's not nil. It will only send using multipart if needed.
// It is equivalent to calling Do with "POST"
func POST(c *httputil.Client, data DataMultipartWriter, v any, url string) error {
	return Do(c, "POST", data, v, url)
}

// PATCH sends a PATCH request using client to the given URL and unmarshals the
// body into v if it's not nil. It will only send using multipart if needed.
// It is equivalent to calling Do with "PATCH"
func PATCH(c *httputil.Client, data DataMultipartWriter, v any, url string) error {
	return Do(c, "PATCH", data, v, url)
}

// Write writes the item into payload_json and the list of files into the
// multipart writer. Write does not close the body.
func Write(body *multipart.Writer, item any, files []File) error {
	// Encode the JSON body first
	w, err := body.CreateFormField("payload_json")
	if err != nil {
		return fmt.Errorf("failed to create bodypart for JSON: %w", err)
	}

	if err := json.EncodeStream(w, item); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	for i, file := range files {
		num := strconv.Itoa(i)

		w, err := body.CreateFormFile("file"+num, file.Name)
		if err != nil {
			return fmt.Errorf("failed to create bodypart for %q: %w", num, err)
		}

		if _, err := io.Copy(w, file.Reader); err != nil {
			return fmt.Errorf("failed to write for file %q: %w", num, err)
		}
	}

	return nil
}
