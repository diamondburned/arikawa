package bot

import (
	"errors"
	"strings"
)

type ErrUnknownCommand struct {
	Prefix  string
	Command string
	Parent  string

	// TODO: list available commands?
	// Here, as a reminder
	ctx []*CommandContext
}

func (err *ErrUnknownCommand) Error() string {
	return UnknownCommandString(err)
}

var UnknownCommandString = func(err *ErrUnknownCommand) string {
	var header = "Unknown command: " + err.Prefix
	if err.Parent != "" {
		header += err.Parent + " " + err.Command
	} else {
		header += err.Command
	}

	return header
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
	Ctx *CommandContext
}

func (err *ErrInvalidUsage) Error() string {
	return InvalidUsageString(err)
}

func (err *ErrInvalidUsage) Unwrap() error {
	return err.Wrap
}

var InvalidUsageString = func(err *ErrInvalidUsage) string {
	if err.Index == 0 {
		return "Invalid usage, error: " + err.Wrap.Error() + "."
	}

	if len(err.Args) == 0 {
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
