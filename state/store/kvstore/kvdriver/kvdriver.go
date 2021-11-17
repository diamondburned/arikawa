// Package kvdriver defines interfaces and abstractions for key-value databases.
package kvdriver

import (
	"errors"
)

// TransactionalDatabase defines a key-value database allowing transactions.
// Databases that don't support transactions can stub the method by returning
// itself, though doing so might compromise data integrity.
type TransactionalDatabase interface {
	Begin(readOnly bool) (Transaction, error)
}

// Transaction defines a transaction within a database.
type Transaction interface {
	// Rollback rolls back the transaction. It is a method that must not fail,
	// and since if it does fail, there isn't a meaningful way for Store to
	// handle it. In most cases, the implementation should just panic.
	Rollback()
	// Commit commits the changes of the transaction into the database. The
	// store implementation will only call this once all the operations are
	// done.
	Commit() error

	Database
}

// EachBreak is a value that asks Database to break out of an Each loop.
var EachBreak = errors.New("break each (not an error)")

// Database defines a nested key-value database API. It is meant to be used
// inside a transaction.
type Database interface {
	// Bucket returns an existing bucket with the given path, or a new bucket if
	// the transaction is not read-only and the bucket doesn't yet exist.
	//
	// The path given to Bucket will be the path relative to the current
	// Database's bucket. For example, the following code:
	//
	//    b, _ = b.Bucket("a", "b")
	//    b, _ = b.Bucket("c", "d")
	//
	// should behave the same as
	//
	//    b, _ = b.Bucket("a", "b", "c", "d")
	//
	// assuming that Bucket will never fail.
	Bucket(keys ...string) (Database, error)
	// Get gets the given key and unmarshals it into the given value, which will
	// always be a pointer to a value.
	Get(key string, v interface{}) error
	// Set sets the given key and value into the database. Marshaling is done by
	// the database implementation. The given value is not necessarily a
	// pointer.
	Set(key string, v interface{}) error
	// Each iterates over each key in the database bucket and unmarshals the
	// value into tmp, similarly to Get. If tmp is nil, then the database must
	// skip unmarshaling the value. The iteration order doesn't have to be
	// ordered. If fn returns EachBreak, then Database must not call fn again.
	Each(tmp interface{}, fn func(k string) error) error
	// EachIsOrdered should return true if the database's Each method iterates
	// in ascending order, ordered lexicographically by keys.
	EachIsOrdered() bool
}
