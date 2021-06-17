package sendpart

import (
	"io"
	"mime/multipart"
	"net/url"
	"strconv"

	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
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
func Do(c *httputil.Client, method string, data DataMultipartWriter, v interface{}, url string) error {
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
func POST(c *httputil.Client, data DataMultipartWriter, v interface{}, url string) error {
	return Do(c, "POST", data, v, url)
}

// PATCH sends a PATCH request using client to the given URL and unmarshals the
// body into v if it's not nil. It will only send using multipart if needed.
// It is equivalent to calling Do with "PATCH"
func PATCH(c *httputil.Client, data DataMultipartWriter, v interface{}, url string) error {
	return Do(c, "PATCH", data, v, url)
}

// Write writes the item into payload_json and the list of files into the
// multipart writer. Write does not close the body.
func Write(body *multipart.Writer, item interface{}, files []File) error {
	// Encode the JSON body first
	w, err := body.CreateFormField("payload_json")
	if err != nil {
		return errors.Wrap(err, "failed to create bodypart for JSON")
	}

	if err := json.EncodeStream(w, item); err != nil {
		return errors.Wrap(err, "failed to encode JSON")
	}

	for i, file := range files {
		num := strconv.Itoa(i)

		w, err := body.CreateFormFile("file"+num, file.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create bodypart for "+num)
		}

		if _, err := io.Copy(w, file.Reader); err != nil {
			return errors.Wrap(err, "failed to write for file "+num)
		}
	}

	return nil
}
