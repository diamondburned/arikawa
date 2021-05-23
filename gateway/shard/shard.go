// Package shard provides sharding capabilities through Manager, found in every
// session.Session.
package shard

// GenerateShardIDs generates an array of ints of 0..(total-1).
func GenerateShardIDs(total int) []int {
	ids := make([]int, total)

	for i := range ids {
		ids[i] = i
	}

	return ids
}

// Error is the error returned if an action on a single shard fails.
type Error struct {
	// ShardID is the id of the shard that returned the error.
	ShardID int
	// Source is the error itself
	Source error
}

func (err *Error) Unwrap() error {
	return err.Source
}

func (err *Error) Error() string {
	panic("implement me")
}
