package arguments

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/utils/bot"
)

// Joined implements ManualParseable, in case you want all arguments but
// joined in a uniform way with spaces.
type Joined string

var _ bot.ManualParser = (*Joined)(nil)

func (j *Joined) ParseContent(args []string) error {
	*j = Joined(strings.Join(args, " "))
	return nil
}
