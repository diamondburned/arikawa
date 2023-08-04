package moreatomic

import (
	"sync/atomic"

	"libdb.so/arikawa/v4/discord"
)

type Snowflake int64

func (s *Snowflake) Get() discord.Snowflake {
	return discord.Snowflake(atomic.LoadInt64((*int64)(s)))
}

func (s *Snowflake) Set(id discord.Snowflake) {
	atomic.StoreInt64((*int64)(s), int64(id))
}
