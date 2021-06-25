package shard

import (
	"context"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/internal/backoff"
	"github.com/pkg/errors"
)

func updateIdentifier(ctx context.Context, id *gateway.Identifier) (url string, err error) {
	botData, err := api.NewClient(id.Token).WithContext(ctx).BotURL()
	if err != nil {
		return "", err
	}

	if botData.Shards < 1 {
		botData.Shards = 1
	}

	id.Shard = &gateway.Shard{0, botData.Shards}

	// Update the burst to be the current given time and reset it back to
	// the default when the given time is reached.
	id.IdentifyGlobalLimit.SetBurst(botData.StartLimit.Remaining)
	resetAt := time.Now().Add(botData.StartLimit.ResetAfter.Duration())
	id.IdentifyGlobalLimit.SetBurstAt(resetAt, botData.StartLimit.Total)

	// Update the maximum number of identify requests allowed per 5s.
	id.IdentifyShortLimit.SetBurst(botData.StartLimit.MaxConcurrency)

	return botData.URL, nil
}

// Manager is the manager responsible for handling all sharding on this
// instance. An instance of Manager must never be copied.
type Manager struct {
	// shards are the *shards.shards managed by this Manager. They are
	// sorted in ascending order by their shard id.
	shards     []ShardState
	gatewayURL string

	mutex sync.RWMutex

	rescaling *rescalingState // nil unless rescaling

	new NewShardFunc
}

type rescalingState struct {
	haltRescale context.CancelFunc
	rescaleDone sync.WaitGroup
}

// NewManager creates a Manager using as many gateways as recommended by
// Discord.
func NewManager(token string, fn NewShardFunc) (*Manager, error) {
	id := gateway.DefaultIdentifier(token)

	url, err := updateIdentifier(context.Background(), id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gateway info")
	}

	return NewIdentifiedManagerWithURL(url, id, fn)
}

// NewIdentifiedManager creates a new Manager using the given
// gateway.Identifier. The total number of shards will be taken from the
// identifier instead of being queried from Discord, but the shard ID will be
// ignored.
//
// This function should rarely be used, since the shard information will be
// queried from Discord if it's required to shard anyway.
func NewIdentifiedManager(data gateway.IdentifyData, fn NewShardFunc) (*Manager, error) {
	// Ensure id.Shard is never nil.
	if data.Shard == nil {
		data.Shard = gateway.DefaultShard
	}

	id := gateway.NewIdentifier(data)

	url, err := updateIdentifier(context.Background(), id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gateway info")
	}

	id.Shard = data.Shard

	return NewIdentifiedManagerWithURL(url, id, fn)
}

// NewIdentifiedManagerWithURL creates a new Manager with the given Identifier
// and gateway URL. It behaves similarly to NewIdentifiedManager.
func NewIdentifiedManagerWithURL(
	url string, id *gateway.Identifier, fn NewShardFunc) (*Manager, error) {

	m := Manager{
		gatewayURL: gateway.AddGatewayParams(url),
		shards:     make([]ShardState, id.Shard.NumShards()),
		new:        fn,
	}

	var err error

	for i := range m.shards {
		data := id.IdentifyData
		data.Shard = &gateway.Shard{i, len(m.shards)}

		m.shards[i] = ShardState{
			ID: gateway.Identifier{
				IdentifyData:        data,
				IdentifyShortLimit:  id.IdentifyShortLimit,
				IdentifyGlobalLimit: id.IdentifyGlobalLimit,
			},
		}

		m.shards[i].Shard, err = fn(&m, &m.shards[i].ID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create shard %d/%d", i, len(m.shards)-1)
		}
	}

	return &m, nil
}

// GatewayURL returns the URL to the gateway. The URL will always have the
// needed gateway parameters appended.
func (m *Manager) GatewayURL() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.gatewayURL
}

// NumShards returns the total number of shards. It is OK for the caller to rely
// on NumShards while they're inside ForEach.
func (m *Manager) NumShards() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.shards)
}

// Shard gets the shard with the given ID.
func (m *Manager) Shard(ix int) Shard {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if ix >= len(m.shards) {
		return nil
	}

	return m.shards[ix]
}

// FromGuildID returns the Shard and the shard ID for the guild with the given
// ID.
func (m *Manager) FromGuildID(guildID discord.GuildID) (shard Shard, ix int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ix = int(uint64(guildID>>22) % uint64(len(m.shards)))
	return m.shards[ix], ix
}

// ForEach calls the given function on each shard from first to last. The caller
// can safely access the number of shards by either asserting Shard to get the
// IdentifyData or call m.NumShards.
func (m *Manager) ForEach(f func(shard Shard)) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, g := range m.shards {
		f(g.Shard)
	}
}

// Open opens all gateways handled by this Manager. If an error occurs, Open
// will attempt to close all previously opened gateways before returning.
func (m *Manager) Open(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return OpenShards(ctx, m.shards)
}

// Close closes all gateways handled by this Manager; it will stop rescaling if
// the manager is currently being rescaled. If an error occurs, Close will
// attempt to close all remaining gateways first, before returning.
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.rescaling != nil {
		m.rescaling.haltRescale()
		m.rescaling.rescaleDone.Wait()

		m.rescaling = nil
	}

	return CloseShards(m.shards)
}

// Rescale rescales the manager asynchronously. The caller MUST NOT call Rescale
// in the constructor function; doing so WILL cause the state to be inconsistent
// and eventually crash and burn and destroy us all.
func (m *Manager) Rescale() {
	go m.rescale()
}

func (m *Manager) rescale() {
	m.mutex.Lock()

	// Exit if we're already rescaling.
	if m.rescaling != nil {
		m.mutex.Unlock()
		return
	}

	// Create a new context to allow the caller to cancel rescaling.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.rescaling = &rescalingState{haltRescale: cancel}
	m.rescaling.rescaleDone.Add(1)
	defer m.rescaling.rescaleDone.Done()

	// Take the old list of shards for ourselves.
	oldShards := m.shards
	m.shards = nil

	m.mutex.Unlock()

	// Close the shards outside the lock. This should be fairly quickly, but it
	// allows the caller to halt rescaling while we're closing or opening the
	// shards.
	CloseShards(oldShards)

	backoffT := backoff.NewTimer(time.Second, 15*time.Minute)
	defer backoffT.Stop()

	for {
		if m.tryRescale(ctx) {
			return
		}

		select {
		case <-backoffT.Next():
			continue
		case <-ctx.Done():
			return
		}
	}
}

// tryRescale attempts once to rescale. It assumes the mutex is unlocked and
// will unlock the mutex itself.
func (m *Manager) tryRescale(ctx context.Context) bool {
	m.mutex.Lock()

	data := m.shards[0].ID.IdentifyData
	newID := gateway.NewIdentifier(data)

	url, err := updateIdentifier(ctx, newID)
	if err != nil {
		m.mutex.Unlock()
		return false
	}

	numShards := newID.Shard.NumShards()
	m.gatewayURL = url

	// Release the mutex early.
	m.mutex.Unlock()

	// Create the shards slice to set after we reacquire the mutex.
	newShards := make([]ShardState, numShards)

	for i := 0; i < numShards; i++ {
		data := newID.IdentifyData
		data.Shard = &gateway.Shard{i, len(m.shards)}

		newShards[i] = ShardState{
			ID: gateway.Identifier{
				IdentifyData:        data,
				IdentifyShortLimit:  newID.IdentifyShortLimit,
				IdentifyGlobalLimit: newID.IdentifyGlobalLimit,
			},
		}

		newShards[i].Shard, err = m.new(m, &newShards[i].ID)
		if err != nil {
			return false
		}
	}

	if err := OpenShards(ctx, newShards); err != nil {
		return false
	}

	m.mutex.Lock()
	m.shards = newShards
	m.rescaling = nil
	m.mutex.Unlock()

	return true
}
