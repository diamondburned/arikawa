package bot

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/bot/shellwords"
)

type argumentValueFn func(string) (reflect.Value, error)

// Parser implements a Parse(string) method for data structures that can be
// used as arguments.
type Parser interface {
	Parse(string) error
}

// ManualParser has a ParseContent(string) method. If the library sees
// this for an argument, it will send all of the arguments (including the
// command) into the method. If used, this should be the only argument followed
// after the Message Create event. Any more and the router will ignore.
type ManualParser interface {
	// $0 will have its prefix trimmed.
	ParseContent([]string) error
}

// RawArguments implements ManualParseable, in case you want to implement a
// custom argument parser. It borrows the library's argument parser.
type RawArguments struct {
	Command   string
	Arguments []string
}

func (r *RawArguments) ParseContent(args []string) error {
	r.Command = args[0]

	if len(args) > 1 {
		r.Arguments = args[1:]
	}

	return nil
}

func (r RawArguments) Arg(n int) string {
	if n < 0 || n >= len(r.Arguments) {
		return ""
	}

	return r.Arguments[n]
}

func (r RawArguments) After(n int) string {
	if n < 0 || n >= len(r.Arguments) {
		return ""
	}

	return strings.Join(r.Arguments[n:], " ")
}

func (r RawArguments) String() string {
	return r.Command + " " + strings.Join(r.Arguments, " ")
}

func (r RawArguments) Length() int {
	return len(r.Arguments)
}

// CustomParser has a CustomParse method, which would be passed in the full
// message content with the prefix trimmed (but not the command). This is used
// for commands that require more advanced parsing than the default CSV reader.
type CustomParser interface {
	CustomParse(content string) error
}

// CustomArguments implements the CustomParser interface, which sets the string
// exactly.
type Content string

func (c *Content) CustomParse(content string) error {
	*c = Content(content)
	return nil
}

// Argument is each argument in a method.
type Argument struct {
	String string
	// Rule: pointer for structs, direct for primitives
	Type reflect.Type

	// indicates if the type is referenced, meaning it's a pointer but not the
	// original call.
	pointer bool

	// if nil, then manual
	fn     argumentValueFn
	manual *reflect.Method
	custom *reflect.Method
}

var ShellwordsEscaper = strings.NewReplacer(
	"\\", "\\\\",
)

var ParseArgs = func(args string) ([]string, error) {
	return shellwords.Parse(args)
}

// nilV, only used to return an error
var nilV = reflect.Value{}

func getArgumentValueFn(t reflect.Type) (*Argument, error) {
	var typeI = t
	var ptr = false

	if t.Kind() != reflect.Ptr {
		typeI = reflect.PtrTo(t)
		ptr = true
	}

	if typeI.Implements(typeIParser) {
		mt, ok := typeI.MethodByName("Parse")
		if !ok {
			panic("BUG: type IParser does not implement Parse")
		}

		avfn := func(input string) (reflect.Value, error) {
			v := reflect.New(t.Elem())

			ret := mt.Func.Call([]reflect.Value{
				v, reflect.ValueOf(input),
			})

			if err := errorReturns(ret); err != nil {
				return nilV, err
			}

			if ptr {
				v = v.Elem()
			}

			return v, nil
		}

		return &Argument{
			String:  t.String(),
			Type:    typeI,
			pointer: ptr,
			fn:      avfn,
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

	return &Argument{
		String: t.String(),
		Type:   t,
		fn:     fn,
	}, nil
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
