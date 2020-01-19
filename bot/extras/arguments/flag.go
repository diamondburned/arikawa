package arguments

import (
	"bytes"
	"flag"
	"io/ioutil"
	"strings"
)

var FlagName = "command"

type FlagSet struct {
	*flag.FlagSet
}

func NewFlagSet() *FlagSet {
	fs := flag.NewFlagSet(FlagName, flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)

	return &FlagSet{fs}
}

func (fs *FlagSet) Usage() string {
	var buf bytes.Buffer

	fs.FlagSet.SetOutput(&buf)
	fs.FlagSet.Usage()
	fs.FlagSet.SetOutput(ioutil.Discard)

	return buf.String()
}

type Flag struct {
	arguments []string
}

func (f *Flag) ParseContent(arguments []string) error {
	// trim the command out
	f.arguments = arguments[1:]
	return nil
}

func (f *Flag) Usage() string {
	return "flags..."
}

func (f *Flag) Args() []string {
	return f.arguments
}

func (f *Flag) Arg(n int) string {
	if n < 0 || n >= len(f.arguments) {
		return ""
	}

	return f.arguments[n]
}

func (f *Flag) String() string {
	return strings.Join(f.arguments, " ")
}

func (f *Flag) With(fs *flag.FlagSet) error {
	return fs.Parse(f.arguments)
}
