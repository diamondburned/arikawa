package bot

import (
	"github.com/diamondburned/arikawa/v3/state"
)

type Parser interface {
	Parse(string) error
}

type Autocompleter interface{}

type State struct {
	*state.State
}
