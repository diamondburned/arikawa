package bot

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
)

type argumentValueFn func(string) (reflect.Value, error)

// Parser implements a Parse(string) method for data structures that can be
// used as arguments.
type Parser interface {
	Parse(string) error
}

// Usager is used in place of the automatically parsed struct name for Parser
// and other interfaces.
type Usager interface {
	Usage() string
}

// ManualParser has a ParseContent(string) method. If the library sees
// this for an argument, it will send all of the arguments into the method. If
// used, this should be the only argument followed after the Message Create
// event. Any more and the router will ignore.
type ManualParser interface {
	// $0 will have its prefix trimmed.
	ParseContent([]string) error
}

// ArgumentParts implements ManualParser, in case you want to parse arguments
// manually. It borrows the library's argument parser.
type ArgumentParts []string

var _ ManualParser = (*ArgumentParts)(nil)

// ParseContent implements ManualParser.
func (r *ArgumentParts) ParseContent(args []string) error {
	*r = args
	return nil
}

func (r ArgumentParts) Arg(n int) string {
	if n < 0 || n >= len(r) {
		return ""
	}
	return r[n]
}

func (r ArgumentParts) After(n int) string {
	if n < 0 || n > len(r) {
		return ""
	}
	return strings.Join(r[n:], " ")
}

func (r ArgumentParts) String() string {
	return strings.Join(r, " ")
}

func (r ArgumentParts) Length() int {
	return len(r)
}

// Usage implements Usager.
func (r ArgumentParts) Usage() string {
	return "strings"
}

// CustomParser has a CustomParse method, which would be passed in the full
// message content with the prefix, command, subcommand and space trimmed. This
// is used for commands that require more advanced parsing than the default
// parser.
type CustomParser interface {
	CustomParse(arguments string) error
}

// RawArguments implements the CustomParser interface, which sets all the
// arguments into it as raw as it could.
type RawArguments string

var _ CustomParser = (*RawArguments)(nil)

func (a *RawArguments) CustomParse(arguments string) error {
	*a = RawArguments(arguments)
	return nil
}

// Argument is each argument in a method.
type Argument struct {
	String string
	// Rule: pointer for structs, direct for primitives
	rtype reflect.Type

	// indicates if the type is referenced, meaning it's a pointer but not the
	// original call.
	pointer bool

	// if nil, then manual
	fn argumentValueFn

	manual func(ManualParser, []string) error
	custom func(CustomParser, string) error
}

// Type returns the argument's reflection type.
func (a *Argument) Type() reflect.Type {
	return a.rtype
}

// CommandOptionType returns the CommandOptionType for this parameter.
// StringOption is returned if the type is not a primitive.
func (a Argument) CommandOptionType() discord.CommandOptionType {
	switch a.rtype.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fallthrough
	case reflect.Float32, reflect.Float64:
		return discord.IntegerOption
	case reflect.Bool:
		return discord.BooleanOption
	case reflect.String:
		return discord.StringOption
	default:
		return discord.StringOption
	}
}

var ShellwordsEscaper = strings.NewReplacer(
	"\\", "\\\\",
)

var (
	// nilV, only used to return an error
	nilV = reflect.Value{}

	trueV  = reflect.ValueOf(true)
	falseV = reflect.ValueOf(false)
)

func newArgument(t reflect.Type, variadic bool) (*Argument, error) {
	// Allow array types if variadic is true.
	if variadic && t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	var typeI = t
	var ptr = false

	if t.Kind() != reflect.Ptr {
		typeI = reflect.PtrTo(t)
		ptr = true
	}

	// This shouldn't be variadic.
	if !variadic && typeI.Implements(typeICusP) {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		return &Argument{
			String:  fromUsager(t),
			rtype:   t,
			pointer: ptr,
			custom:  CustomParser.CustomParse,
		}, nil
	}

	// This shouldn't be variadic either.
	if !variadic && typeI.Implements(typeIManP) {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		return &Argument{
			String:  fromUsager(t),
			rtype:   t,
			pointer: ptr,
			manual:  ManualParser.ParseContent,
		}, nil
	}

	if typeI.Implements(typeIParser) {
		mt, ok := typeI.MethodByName("Parse")
		if !ok {
			panic("BUG: type IParser does not implement Parse")
		}

		avfn := func(input string) (reflect.Value, error) {
			v := reflect.New(typeI.Elem())

			ret := mt.Func.Call([]reflect.Value{
				v, reflect.ValueOf(input),
			})

			_, err := errorReturns(ret)
			if err != nil {
				return nilV, err
			}

			if ptr {
				v = v.Elem()
			}

			return v, nil
		}

		return &Argument{
			String:  fromUsager(typeI),
			rtype:   typeI,
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

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fn = func(s string) (reflect.Value, error) {
			i, err := strconv.ParseInt(s, 10, 64)
			return quickRet(i, err, t)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
			case "True", "TRUE", "true", "T", "t", "yes", "y", "Y", "1":
				return trueV, nil
			case "False", "FALSE", "false", "F", "f", "no", "n", "N", "0":
				return falseV, nil
			default:
				return nilV, errors.New("invalid bool [true|false]")
			}
		}
	}

	if fn == nil {
		return nil, errors.New("invalid type: " + t.String())
	}

	return &Argument{
		String: fromUsager(t),
		rtype:  t,
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

func fromUsager(typeI reflect.Type) string {
	if typeI.Implements(typeIUsager) {
		mt, _ := typeI.MethodByName("Usage")

		vs := mt.Func.Call([]reflect.Value{reflect.New(typeI).Elem()})
		return vs[0].String()
	}

	s := strings.Split(typeI.String(), ".")
	return s[len(s)-1]
}
