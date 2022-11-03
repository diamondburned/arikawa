// Package json allows for different implementations of JSON serializing, as
// well as extra optional types needed.
package json

import (
	"encoding/json"
	"io"
	"reflect"
	"unsafe"
)

type Driver interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error

	DecodeStream(r io.Reader, v interface{}) error
	EncodeStream(w io.Writer, v interface{}) error
}

type DefaultDriver struct{}

func (d DefaultDriver) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (d DefaultDriver) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (d DefaultDriver) DecodeStream(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func (d DefaultDriver) EncodeStream(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// Default is the default JSON driver, which uses encoding/json.
var Default Driver = DefaultDriver{}

// Marshal uses the default driver.
func Marshal(v interface{}) ([]byte, error) {
	return Default.Marshal(v)
}

// Unmarshal uses the default driver.
func Unmarshal(data []byte, v interface{}) error {
	return Default.Unmarshal(data, v)
}

// DecodeStream uses the default driver.
func DecodeStream(r io.Reader, v interface{}) error {
	return Default.DecodeStream(r, v)
}

// EncodeStream uses the default driver.
func EncodeStream(w io.Writer, v interface{}) error {
	return Default.EncodeStream(w, v)
}

// PartialUnmarshal partially unmarshals the JSON in b onto v. Fields that
// cannot be unmarshaled will be left as their zero values. Fields that can be
// marshaled will be unmarshaled and its name will be added to the returned
// slice.
//
// Only use this for the most cursed of JSONs, such as ones coming from Discord.
// Try not to use this as much as possible.
func PartialUnmarshal(b []byte, v interface{}) []error {
	var errs []error

	dstv := reflect.Indirect(reflect.ValueOf(v))
	dstt := dstv.Type()

	// ptrVal will be used by reflect to store our temporary object. This allows
	// us to free up n heap allocations just for one pointer.
	var ptrVal struct {
		_ *struct{}
	}

	dstfields := dstt.NumField()
	for i := 0; i < dstfields; i++ {
		dstfield := dstt.Field(i)

		// Create us a custom struct with this one field, except its type is a
		// pointer type. We prefer to do this over parsing JSON's tags.
		fake := reflect.NewAt(reflect.StructOf([]reflect.StructField{
			{
				Name: dstfield.Name,
				Type: reflect.PtrTo(dstfield.Type),
				Tag:  dstfield.Tag,
			},
		}), unsafe.Pointer(&ptrVal))

		// We can use this pointer to set the value of the field.
		fake.Elem().Field(0).Set(dstv.Field(i).Addr())

		// Unmarshal into this struct.
		if err := json.Unmarshal(b, fake.Interface()); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}
