package httputil

import (
	"net/url"
	"sync"

	"github.com/gorilla/schema"
)

// SchemaEncoder expects the encoder to read the "schema" tags.
type SchemaEncoder interface {
	Encode(src interface{}) (url.Values, error)
}

type DefaultSchema struct {
	once sync.Once
	*schema.Encoder
}

var _ SchemaEncoder = (*DefaultSchema)(nil)

func (d *DefaultSchema) Encode(src interface{}) (url.Values, error) {
	if d.Encoder == nil {
		d.once.Do(func() {
			d.Encoder = schema.NewEncoder()
		})
	}

	var v = url.Values{}
	return v, d.Encoder.Encode(src, v)
}
