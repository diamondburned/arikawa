package arguments

import (
	"bytes"
	"flag"
	"io/ioutil"
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

type Flag []string

func (f *Flag) ParseContent(arguments []string) error {
	*f = arguments
	return nil
}

func (f Flag) Usage() string {
	return "[flags] arguments"
}

func (f Flag) Args() []string {
	return f
}

func (f Flag) With(fs *flag.FlagSet) error {
	return fs.Parse(f)
}
