package gateway

type Shard [2]int

func DefaultShard() *Shard {
	var s = Shard([2]int{0, 1})
	return &s
}

func (s Shard) ShardID() int {
	return s[0]
}

func (s Shard) NumShards() int {
	return s[1]
}
