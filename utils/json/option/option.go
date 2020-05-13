// Package option provides the ability to create omittable primitives.
// This is accomplished by pointerrizing common primitive types so that they may
// assume a nil value, which is considered as omitted by encoding/json.
// To generate pointerrized primitives, there are helper functions NewT() for
// each option type.
package option
