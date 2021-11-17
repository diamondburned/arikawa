package kvdriver

import (
	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

// ByteCodec describes a pair of marshaler and unmarshaler methods for
// ByteDatabase and BasicByteDatabase to use.
type ByteCodec interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}

// JSONByteCodec is a ByteCodec that uses JSON for marshaling and unmarshaling.
var JSONByteCodec = jsonCodec{}

type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

type byteDatabase interface {
	// Get gets the given key and returns the byte slice.
	Get(key string) ([]byte, error)
	// Set sets the given key and byte slice value into the database.
	Set(key string, v []byte) error
	// EachIsOrdered works the same as Database.
	EachIsOrdered() bool
}

// ByteDatabase defines a database that can only store and retrieve bytes. It is
// used for wrapping using WrapByteDatabase to form a Database.
type ByteDatabase interface {
	// Bucket should work exactly the same as Database's.
	Bucket(keys ...string) (ByteDatabase, error)
	// Get gets the given key and returns the byte slice.
	Get(key string) ([]byte, error)
	// Set sets the given key and byte slice value into the database.
	Set(key string, v []byte) error
	// Each should behave similarly to Database's, except the bytes value is
	// passed directly into the callback instead of unmarshaled indirectly.
	Each(fn func(k string, v []byte) error) error
	// EachIsOrdered works the same as Database.
	EachIsOrdered() bool
}

// BasicByteDatabase is the basic variant of ByteDatabase.
type BasicByteDatabase interface {
	// Bucket should work exactly the same as Database's.
	Bucket(keys ...string) (ByteDatabase, error)
	// Get gets the given key and returns the byte slice.
	Get(key string) ([]byte, error)
	// Set sets the given key and byte slice value into the database.
	Set(key string, v []byte) error
	// Each should behave similarly to BasicDatabase's.
	Each(fn func(k string) error) error
	// EachIsOrdered works the same as Database.
	EachIsOrdered() bool
}

type wrappedByteDatabase struct {
	wrappedSmallByteDatabase
}

// WrapByteDatabase wraps the given ByteDatabase into a database.
func WrapByteDatabase(bytedb ByteDatabase, codec ByteCodec) Database {
	return wrappedByteDatabase{
		wrappedSmallByteDatabase: wrapSmallByteDatabase(bytedb, codec),
	}
}

func (w wrappedByteDatabase) EachIsOrdered() bool {
	return w.byteDatabase.(ByteDatabase).EachIsOrdered()
}

func (w wrappedByteDatabase) Bucket(keys ...string) (Database, error) {
	d, err := w.byteDatabase.(ByteDatabase).Bucket(keys...)
	if err != nil {
		return nil, err
	}
	w.byteDatabase = d
	return w, nil
}

func (w wrappedByteDatabase) Each(tmp interface{}, fn func(k string) error) error {
	db := w.byteDatabase.(ByteDatabase)
	return db.Each(func(k string, b []byte) error {
		if err := w.Codec.Unmarshal(b, tmp); err != nil {
			return errors.Wrap(err, "error unmarshaling ByteDatabase value")
		}
		return fn(k)
	})
}

type wrappedBasicByteDatabase struct {
	wrappedSmallByteDatabase
}

// WrapBasicByteDatabase wraps a BasicByteDatabase into a BasicDatabase. The
// user should use WrapByteDatabase to wrap the given BasicDatabase into a
// Database.
func WrapBasicByteDatabase(basic BasicByteDatabase, codec ByteCodec) BasicDatabase {
	return wrappedBasicByteDatabase{
		wrappedSmallByteDatabase: wrapSmallByteDatabase(basic, codec),
	}
}

func (w wrappedBasicByteDatabase) Each(fn func(k string) error) error {
	return w.byteDatabase.(BasicByteDatabase).Each(fn)
}

type wrappedSmallByteDatabase struct {
	byteDatabase
	Codec ByteCodec
}

func wrapSmallByteDatabase(basic byteDatabase, codec ByteCodec) wrappedSmallByteDatabase {
	return wrappedSmallByteDatabase{
		byteDatabase: basic,
		Codec:        codec,
	}
}

func (w wrappedSmallByteDatabase) Get(key string, v interface{}) error {
	b, err := w.byteDatabase.Get(key)
	if err != nil {
		return err
	}
	if err := w.Codec.Unmarshal(b, v); err != nil {
		return errors.Wrap(err, "error unmarshaling ByteDatabase value")
	}
	return nil
}

func (w wrappedSmallByteDatabase) Set(key string, v interface{}) error {
	b, err := w.Codec.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling ByteDatabase value")
	}
	if err := w.byteDatabase.Set(key, b); err != nil {
		return err
	}
	return nil
}
