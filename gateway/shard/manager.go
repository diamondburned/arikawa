package shard

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
)

// Manager is the manager responsible for handling all sharding
type Manager struct {
	// Gateways are the *gateway.Gateways managed by this Manager.
	// They are sorted in ascending order by their shard id.
	Gateways []*gateway.Gateway

	// TotalShards is the total number of shards.
	// This may be higher than len(Gateways), if other shards are running in
	// a different process/on a different machine.
	TotalShards int

	// OnShardingRequired is the function called, if Discord closes any of the
	// gateways with a 4011 close code.
	// By default, if len(Gateways) == TotalShards, the Manager will
	// automatically rescale using the recommended number of shards as received
	// from Discord.
	OnShardingRequired func()
}

func NewManagerFromGateways(totalShards int, gateways ...*gateway.Gateway) *Manager {
	panic("implement me!")
}

func NewManagerFromShardIDs(token string, totalShards int, shardIDs ...int) *Manager {
	panic("implement me!")
}

// NewAutomaticManager creates as many Gateways as recommended by Discord.
func NewAutomaticManager(token string) *Manager {
	panic("implement me!")
}

// FromShardID returns the *gateway.Gateway with the given shard id, or nil if
// the shard manager has no gateway with the given id.
func (m *Manager) FromShardID(shardID int) *gateway.Gateway {
	panic("uses sort.Search to find the correct shard")
}

// FromGuildID returns the *gateway.Gateway managing the guild with the passed
// ID, or nil if this Manager does not manage this guild.
func (m *Manager) FromGuildID(guildID discord.GuildID) *gateway.Gateway {
	return m.FromShardID(int(uint64(guildID>>22) % uint64(m.TotalShards)))
}

// Apply applies the given function to all gateways handled by this Manager.
// If the function returns an error, it will return, without applying the
// function to the remaining Gateways.
func (m *Manager) Apply(f func(g *gateway.Gateway) error) error {
	for _, g := range m.Gateways {
		if err := f(g); err != nil {
			return err
		}
	}

	return nil
}

// Open opens all Gateways handled by this Manager.
func (m *Manager) Open() error {
	return m.Apply(func(g *gateway.Gateway) error { return g.Open() })
}

// Close closes all Gateways handled by this Manager.
func (m *Manager) Close() error {
	return m.Apply(func(g *gateway.Gateway) error { return g.Close() })
}

// UpdateStatus updates the status of all Gateways handled by this Manager.
func (m *Manager) UpdateStatus(d gateway.UpdateStatusData) error {
	return m.Apply(func(g *gateway.Gateway) error { return g.UpdateStatus(d) })
}

func (m *Manager) RequestGuildMembers(d gateway.RequestGuildMembersData) error {
	return m.FromGuildID(d.GuildID[0]).RequestGuildMembers(d)
}
