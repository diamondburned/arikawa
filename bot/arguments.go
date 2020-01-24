package bot

import (
	"errors"
	"reflect"
	"strconv"
)

type argumentValueFn func(string) (reflect.Value, error)

// Parseable implements a Parse(string) method for data structures that can be
// used as arguments.
type Parseable interface {
	Parse(string) error
}

// ManaulParseable implements a ParseContent(string) method. If the library sees
// this for an argument, it will send all of the arguments (including the
// command) into the method. If used, this should be the only argument followed
// after the Message Create event. Any more and the router will ignore.
type ManualParseable interface {
	// $0 will have its prefix trimmed.
	ParseContent([]string) error
}

// RawArguments implements ManualParseable, in case you want to implement a
// custom argument parser. It borrows the library's argument parser.
type RawArguments struct {
	Arguments []string
}

func (r *RawArguments) ParseContent(args []string) error {
	r.Arguments = args
	return nil
}

// Argument is each argument in a method.
type Argument struct {
	String string
	Type   reflect.Type
	fn     argumentValueFn
}

// nilV, only used to return an error
var nilV = reflect.Value{}

func getArgumentValueFn(t reflect.Type) (argumentValueFn, error) {
	if t.Implements(typeIParser) {
		mt, ok := t.MethodByName("Parse")
		if !ok {
			panic("BUG: type IParser does not implement Parse")
		}

		return func(input string) (reflect.Value, error) {
			v := reflect.New(t.Elem())

			ret := mt.Func.Call([]reflect.Value{
				v, reflect.ValueOf(input),
			})

			if err := errorReturns(ret); err != nil {
				return nilV, err
			}

			return v, nil
		}, nil
	}

	var fn argumentValueFn

	switch t.Kind() {
	case reflect.String:
		fn = func(s string) (reflect.Value, error) {
			return reflect.ValueOf(s), nil
		}

	case reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64:

		fn = func(s string) (reflect.Value, error) {
			i, err := strconv.ParseInt(s, 10, 64)
			return quickRet(i, err, t)
		}

	case reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64:

		fn = func(s string) (reflect.Value, error) {
			u, err := strconv.ParseUint(s, 10, 64)
			return quickRet(u, err, t)
		}

	case reflect.Float32, reflect.Float64:
		fn = func(s string) (reflect.Value, error) {
			f, err := strconv.ParseFloat(s, 64)
			return quickRet(f, err, t)
		}

	case reflect.Bool:
		fn = func(s string) (reflect.Value, error) {
			switch s {
			case "true", "yes", "y", "Y", "1":
				return reflect.ValueOf(true), nil
			case "false", "no", "n", "N", "0":
				return reflect.ValueOf(false), nil
			default:
				return nilV, errors.New("invalid bool [true/false]")
			}
		}
	}

	if fn == nil {
		return nil, errors.New("invalid type: " + t.String())
	}

	return fn, nil
}

func quickRet(v interface{}, err error, t reflect.Type) (reflect.Value, error) {
	if err != nil {
		return nilV, err
	}

	rv := reflect.ValueOf(v)

	if t == nil {
		return rv, nil
	}

	return rv.Convert(t), nil
}
