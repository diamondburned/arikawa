package api

import (
	"fmt"
)

type ErrOverbound struct {
	Count int
	Max   int

	Thing string
}

var _ error = (*ErrOverbound)(nil)

func (e ErrOverbound) Error() string {
	if e.Thing == "" {
		return fmt.Sprintf("Overbound error: %d > %d", e.Count, e.Max)
	}

	return fmt.Sprintf(e.Thing+" overbound: %d > %d", e.Count, e.Max)
}
