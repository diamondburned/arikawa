package mulipartutil

import (
	"io"
	"mime/multipart"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/utils/json"
)

func WriteMultipart(body *multipart.Writer, item interface{}, files []api.SendMessageFile) error {
	defer body.Close()

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
