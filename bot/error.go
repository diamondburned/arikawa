package bot

import (
	"errors"
	"strings"
)

type ErrUnknownCommand struct {
	Parts  []string // max len 2
	Subcmd *Subcommand
}

func (err *ErrUnknownCommand) Error() string {
	if len(err.Parts) > 2 {
		err.Parts = err.Parts[:2]
	}
	return UnknownCommandString(err)
}

var UnknownCommandString = func(err *ErrUnknownCommand) string {
	return "Unknown command: " + strings.Join(err.Parts, " ")
}

var (
	ErrTooManyArgs   = errors.New("Too many arguments given")
	ErrNotEnoughArgs = errors.New("Not enough arguments given")
)

type ErrInvalidUsage struct {
	Prefix string
	Args   []string
	Index  int
	Wrap   error

	// TODO: usage generator?
	// Here, as a reminder
	Ctx *MethodContext
}

func (err *ErrInvalidUsage) Error() string {
	return InvalidUsageString(err)
}

func (err *ErrInvalidUsage) Unwrap() error {
	return err.Wrap
}

var InvalidUsageString = func(err *ErrInvalidUsage) string {
	if err.Index == 0 && err.Wrap != nil {
		return "Invalid usage, error: " + err.Wrap.Error() + "."
	}

	if err.Index == 0 || len(err.Args) == 0 {
		return "Missing arguments. Refer to help."
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
