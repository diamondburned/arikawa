package kvdriver

import "strings"

// BasicDatabase defines a flat key-value database API. It is useful for
// key-value databases that don't support nesting, such as BadgerDB. Most users
// should use WrapBasicDatabase to wrap the BasicDatabase and create a full
// Database.
type BasicDatabase interface {
	// Get works the same as Database.
	Get(key string, v interface{}) error
	// Set works the same as Database.
	Set(key string, v interface{}) error
	// Each iterates over each key. The database must allow calling Get inside
	// the fn callback.
	Each(fn func(k string) error) error
	// EachIsOrdered works the same as Database.
	EachIsOrdered() bool
}

type wrappedBasicDatabase struct {
	BasicDatabase
	// Delimiter is the default delimiter byte to use. The given keys will be
	// sanitized of this delimiter. The default is a null byte.
	delimiter string
	// state
	prefix string
}

var _ Database = (*wrappedBasicDatabase)(nil)

// WrapBasicDatabase wraps the given basic database to provide an emulated
// nested database. If no delimiter is given, then a single null byte is used.
func WrapBasicDatabase(basic BasicDatabase, delimiter ...byte) Database {
	return wrapBasicDatabase(basic, delimiter)
}

func wrapBasicDatabase(basic BasicDatabase, delimiter []byte) wrappedBasicDatabase {
	if len(delimiter) == 0 {
		delimiter = []byte{0}
	}

	return wrappedBasicDatabase{
		BasicDatabase: basic,
		delimiter:     string(delimiter),
	}
}

func (b wrappedBasicDatabase) EachIsOrdered() bool {
	return b.BasicDatabase.EachIsOrdered()
}

// Bucket returns a copy of wrappedBasicDatabase with a new prefix.
func (b wrappedBasicDatabase) Bucket(keys ...string) (Database, error) {
	b.prefix += strings.Join(keys, b.delimiter) + b.delimiter
	return b, nil
}

// Get gets the given key into v.
func (b wrappedBasicDatabase) Get(key string, v interface{}) error {
	return b.BasicDatabase.Get(b.prefix+key, v)
}

// Sets sets the given key into v.
func (b wrappedBasicDatabase) Set(key string, v interface{}) error {
	return b.BasicDatabase.Set(b.prefix+key, v)
}

// Each wraps around the BasicDatabase by using its key-only Each API.
func (b wrappedBasicDatabase) Each(tmp interface{}, fn func(k string) error) error {
	return b.BasicDatabase.Each(func(k string) error {
		key := strings.TrimPrefix(k, b.prefix)
		if strings.Contains(key, b.delimiter) {
			// Skip, since is nested bucket.
			return nil
		}

		return b.BasicDatabase.Get(k, tmp)
	})
}
