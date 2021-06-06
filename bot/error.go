package bot

import (
	"errors"
	"fmt"
	"strings"
)

type UnknownCommandError struct {
	Subcmd *Subcommand
	Parts  []string // max len 2
}

func newErrUnknownCommand(s *Subcommand, parts []string) error {
	if len(parts) > 2 {
		parts = parts[:2]
	}

	return &UnknownCommandError{
		Parts:  parts,
		Subcmd: s,
	}
}

func (err *UnknownCommandError) Error() string {
	return UnknownCommandString(err)
}

var UnknownCommandString = func(err *UnknownCommandError) string {
	// Subcommand check.
	if err.Subcmd.StructName == "" || len(err.Parts) < 2 {
		return "unknown command: " + err.Parts[0] + "."
	}

	return fmt.Sprintf("unknown %s subcommand: %s.", err.Parts[0], err.Parts[1])
}

var (
	ErrTooManyArgs   = errors.New("too many arguments given")
	ErrNotEnoughArgs = errors.New("not enough arguments given")
)

type InvalidUsageError struct {
	Wrap   error
	Ctx    *MethodContext
	Prefix string
	Args   []string
	Index  int
}

func (err *InvalidUsageError) Error() string {
	return InvalidUsageString(err)
}

func (err *InvalidUsageError) Unwrap() error {
	return err.Wrap
}

var InvalidUsageString = func(err *InvalidUsageError) string {
	if err.Index == 0 && err.Wrap != nil {
		return "invalid usage, error: " + err.Wrap.Error() + "."
	}

	if err.Index == 0 || len(err.Args) == 0 {
		return "missing arguments. Refer to help."
	}

	body := "Invalid usage at " +
		// Write the prefix.
		err.Prefix +
		// Write the first part
		strings.Join(err.Args[:err.Index], " ") +
		// Write the wrong part
		" __" + err.Args[err.Index] + "__ " +
		// Write the last part
		strings.Join(err.Args[err.Index+1:], " ")

	if err.Wrap != nil {
		body += "\nError: " + err.Wrap.Error() + "."
	}

	return body
}
