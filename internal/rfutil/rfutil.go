package rfutil

import (
	"errors"
	"reflect"
)

func StructValue(v interface{}) (reflect.Value, reflect.Type, error) {
	rv := reflect.ValueOf(v)
	return StructRValue(rv)
}

func StructRValue(rv reflect.Value) (reflect.Value, reflect.Type, error) {
	rt := rv.Type()
	if rt.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, errors.New("v is not a pointer")
	}

	rv = rv.Elem()
	rt = rt.Elem()
	if rt.Kind() != reflect.Struct {
		return reflect.Value{}, nil, errors.New("v is not a pointer to a struct")
	}

	return rv, rt, nil
}
