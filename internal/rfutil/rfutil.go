package rfutil

import (
	"errors"
	"math/bits"
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

// KindBits works on int*, uint* and float* only.
func KindBits(k reflect.Kind) int {
	switch k {
	case reflect.Int, reflect.Uint:
		return bits.UintSize
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 32
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 64
	default:
		panic("unknown kind " + k.String())
	}
}
