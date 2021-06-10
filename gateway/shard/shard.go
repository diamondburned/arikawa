package shard

import (
	"context"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/pkg/errors"
)

// Shard defines a shard gateway interface that the shard manager can use.
type Shard interface {
	Open(context.Context) error
	Close() error
}

// NewShardFunc is the constructor to create a new gateway. For examples, see
// package session and state's. The constructor must manually connect the
// Manager's Rescale method appropriately.
//
// A new Gateway must not open any background resources until OpenCtx is called;
// if the gateway has never been opened, its Close method will never be called.
// During callback, the Manager is not locked, so the callback can use Manager's
// methods without deadlocking.
type NewShardFunc func(m *Manager, id *gateway.Identifier) (Shard, error)

// NewGatewayShardFunc wraps around NewGatewayShard to be compatible with
// NewShardFunc.
var NewGatewayShardFunc NewShardFunc = func(m *Manager, id *gateway.Identifier) (Shard, error) {
	return NewGatewayShard(m, id), nil
}

// NewGatewayShard creates a new gateway that's plugged into the shard manager.
func NewGatewayShard(m *Manager, id *gateway.Identifier) *gateway.Gateway {
	gw := gateway.NewCustomIdentifiedGateway(m.GatewayURL(), id)
	gw.OnShardingRequired(m.Rescale)
	return gw
}

// ShardState wraps around the Gateway interface to provide additional state.
type ShardState struct {
	Shard
	// This is a bit wasteful: 2 constant pointers are stored here, and they
	// waste GC cycles. This is unavoidable, however, since the API has to take
	// in a pointer to Identifier, not IdentifyData. This is to ensure rescales
	// are consistent.
	ID     gateway.Identifier
	Opened bool
}

// ShardID returns the shard state's shard ID.
func (state ShardState) ShardID() int {
	return state.ID.Shard.ShardID()
}

// OpenShards opens the gateways of the given list of shard states.
func OpenShards(ctx context.Context, shards []ShardState) error {
	for i, shard := range shards {
		if err := shard.Open(ctx); err != nil {
			CloseShards(shards)
			return errors.Wrapf(err, "failed to open shard %d/%d", i, len(shards)-1)
		}

		// Mark as opened so we can close them.
		shards[i].Opened = true
	}

	return nil
}

// CloseShards closes the gateways of the given list of shard states.
func CloseShards(shards []ShardState) error {
	var lastError error

	for i, gw := range shards {
		if gw.Opened {
			if err := gw.Close(); err != nil {
				lastError = err
			}

			shards[i].Opened = false
		}
	}

	return lastError
}
