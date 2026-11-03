// Package option provides the ability to create omittable primitives.
// This is accomplished by pointerrizing common primitive types so that they may
// assume a nil value, which is considered as omitted by encoding/json.
// To generate pointerrized primitives, there are helper functions NewT() for
// each option type.
package option

// Optional wraps a type to make it omittable.
type Optional[T any] *T

// Some creates a new Optional with the value of the passed type.
func Some[T any](v T) Optional[T] {
	return &v
}

// PtrTo creates a pointer to the passed value.
// It is like [Some], except it returns *T directly rather than Optional[T].
func PtrTo[T any](v T) *T {
	return &v
}
