package shard

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Manager is the manager responsible for handling all sharding on this
// instance.
type Manager struct {
	// gateways are the *gateways.gateways managed by this Manager. They are
	// sorted in ascending order by their shard id.
	gateways []*gateway.Gateway
	mutex    *sync.RWMutex

	// Events is the channel all gateways send their event in.
	Events chan interface{}

	// NumShards is the total number of shards.
	// This may be higher than len(gateways), if other shards are running in
	// a different process/on a different machine.
	NumShards int

	// OnShardingRequired is the function called, if Discord closes any of the
	// gateways with a 4011 close code.
	//
	// By default the Manager was created using NewManager, the manager will
	// automatically rescale using the recommended number of shards as received
	// from Discord. In any other case the Manager will close all gateway
	// connections, unless this function is replaced by a custom one.
	//
	// If you are using a cache like the State does, you need to wipe that
	// cache before reconnecting to the gateway, as some cached object may be
	// outdated.
	OnShardingRequired     func() *Manager
	onShardingRequiredExec *moreatomic.Bool
}

// NewManager creates a Manager using as many gateways as recommended by
// Discord.
func NewManager(token string) (*Manager, error) {
	botData, err := gateway.BotURL(token)
	if err != nil {
		return nil, err
	}

	param := url.Values{
		"v":        {gateway.Version},
		"encoding": {gateway.Encoding},
	}

	id := gateway.DefaultIdentifier(token)

	resetAt := time.Now().Add(botData.StartLimit.ResetAfter.Duration())

	// Update the burst to be the current given time and reset it back to
	// the default when the given time is reached.
	id.IdentifyGlobalLimit.SetBurst(botData.StartLimit.Remaining)
	id.IdentifyGlobalLimit.SetBurstAt(resetAt, botData.StartLimit.Total)

	// Update the maximum number of identify requests allowed per 5s.
	id.IdentifyShortLimit.SetBurst(botData.StartLimit.MaxConcurrency)

	gatewayURL := botData.URL + "?" + param.Encode()

	m := newIdentifiedManager(gatewayURL, id, botData.Shards, GenerateShardIDs(botData.Shards)...)

	m.OnShardingRequired = func() *Manager {
		m, err := NewManager(token)
		if err != nil {
			return nil
		}

		return m
	}

	return m, nil
}

// NewIdentifiedManager creates a new Manager using the passed url and the
// passed gateway.Identifier. The shard information stored on the passed
// identifier will be ignored. Instead totalShards and shardIDs will be used.
func NewIdentifiedManager(id *gateway.Identifier, totalShards int, shardIDs ...int) (*Manager, error) {
	botData, err := gateway.BotURL(id.Token)
	if err != nil {
		return nil, err
	}

	param := url.Values{
		"v":        {gateway.Version},
		"encoding": {gateway.Encoding},
	}

	resetAt := time.Now().Add(botData.StartLimit.ResetAfter.Duration())

	// Update the burst to be the current given time and reset it back to
	// the default when the given time is reached.
	id.IdentifyGlobalLimit.SetBurst(botData.StartLimit.Remaining)
	id.IdentifyGlobalLimit.SetBurstAt(resetAt, botData.StartLimit.Total)

	// Update the maximum number of identify requests allowed per 5s.
	id.IdentifyShortLimit.SetBurst(botData.StartLimit.MaxConcurrency)

	gatewayURL := botData.URL + "?" + param.Encode()
	return newIdentifiedManager(gatewayURL, id, totalShards, shardIDs...), nil
}

func newIdentifiedManager(
	gatewayURL string, id *gateway.Identifier, totalShards int, shardIDs ...int) *Manager {

	gateways := make([]*gateway.Gateway, len(shardIDs))

	for i, shardID := range shardIDs {
		id.SetShard(shardID, totalShards)
		idCp := *id

		gateways[i] = gateway.NewCustomIdentifiedGateway(gatewayURL, &idCp)
	}

	return NewManagerWithGateways(gateways...)
}

// NewManagerWithShardIDs creates a new Manager using the passed token
// to create len(shardIDs) shards with the given ids.
func NewManagerWithShardIDs(token string, totalShards int, shardIDs ...int) (*Manager, error) {
	return NewIdentifiedManager(gateway.DefaultIdentifier(token), totalShards, shardIDs...)
}

// NewManagerWithGateways creates a new Manager from the given
// *gateways.gateways.
func NewManagerWithGateways(gateways ...*gateway.Gateway) *Manager {
	m := &Manager{
		gateways:               gateways,
		Events:                 make(chan interface{}),
		NumShards:              gateways[0].Identifier.Shard.NumShards(),
		onShardingRequiredExec: new(moreatomic.Bool),
	}

	for _, g := range m.gateways {
		g.Events = m.Events
		g.OnScalingRequired = m.onGatewayScalingRequired
	}

	return m
}

// FromShardID returns the *gateway.Gateway with the given shard id, or nil if
// the shard manager has no gateways with the given id.
func (m *Manager) FromShardID(shardID int) *gateway.Gateway {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	i := sort.Search(len(m.gateways), func(i int) bool {
		return m.gateways[i].Identifier.Shard.ShardID() >= shardID
	})

	if i < len(m.gateways) && m.gateways[i].Identifier.Shard.ShardID() == shardID {
		return m.gateways[i]
	}

	return nil
}

// FromGuildID returns the *gateway.Gateway managing the guild with the passed
// ID, or nil if this Manager does not manage this guild.
func (m *Manager) FromGuildID(guildID discord.GuildID) *gateway.Gateway {
	return m.FromShardID(int(uint64(guildID>>22) % uint64(m.NumShards)))
}

// Apply applies the given function to all gateways handled by this Manager.
// If the function returns an error, it will return, without applying the
// function to the remaining gateways.
func (m *Manager) Apply(f func(g *gateway.Gateway) error) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, g := range m.gateways {
		if err := f(g); err != nil {
			return err
		}
	}

	return nil
}

// Gateways returns the gateways managed by this Manager.
func (m *Manager) Gateways() []*gateway.Gateway {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cp := make([]*gateway.Gateway, len(m.gateways))
	copy(cp, m.gateways)

	return cp
}

// Open opens all gateways handled by this Manager.
func (m *Manager) Open() error {
	return m.Apply(func(g *gateway.Gateway) error { return g.Open() })
}

// Close closes all gateways handled by this Manager.
func (m *Manager) Close() error {
	return m.Apply(func(g *gateway.Gateway) error { return g.Close() })
}

// Pause pauses all gateways managed by this Manager.
func (m *Manager) Pause() error {
	return m.Apply(func(g *gateway.Gateway) error { return g.Pause() })
}

// UpdateStatus updates the status of all gateways handled by this Manager.
func (m *Manager) UpdateStatus(d gateway.UpdateStatusData) error {
	return m.Apply(func(g *gateway.Gateway) error { return g.UpdateStatus(d) })
}

func (m *Manager) RequestGuildMembers(d gateway.RequestGuildMembersData) error {
	return m.FromGuildID(d.GuildID[0]).RequestGuildMembers(d)
}

// onGatewayScalingRequired is the function stored as Gateway.OnScalingRequired
// in every of the Manager's gateways.
func (m *Manager) onGatewayScalingRequired() {
	if m.onShardingRequiredExec.CompareAndSwap(false) {
		m.Close()

		if m.OnShardingRequired == nil {
			return
		}

		m.mutex.Lock()
		defer m.mutex.Unlock()

		newM := m.OnShardingRequired()
		if newM == nil {
			return
		}

		*m = *newM
	}
}
