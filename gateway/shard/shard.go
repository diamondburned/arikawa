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
