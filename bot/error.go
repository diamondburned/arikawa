package bot

import (
	"strings"
)

type ErrUnknownCommand struct {
	Command string
	Parent  string

	Prefix string

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

type ErrInvalidUsage struct {
	Args   []string
	Prefix string

	Index int
	Err   string

	// TODO: usage generator?
	// Here, as a reminder
	Ctx *CommandContext
}

func (err *ErrInvalidUsage) Error() string {
	return InvalidUsageString(err)
}

var InvalidUsageString = func(err *ErrInvalidUsage) string {
	if err.Index == 0 {
		return "Invalid usage, error: " + err.Err
	}

	if len(err.Args) == 0 {
		return "Missing arguments. Refer to help."
	}

	body := "Invalid usage at " + err.Prefix

	// Write the first part
	body += strings.Join(err.Args[:err.Index], " ")

	// Write the wrong part
	body += " __" + err.Args[err.Index] + "__ "

	// Write the last part
	body += strings.Join(err.Args[err.Index+1:], " ")

	if err.Err != "" {
		body += "\nError: " + err.Err
	}

	return body
}
