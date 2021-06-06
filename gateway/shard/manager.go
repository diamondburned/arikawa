package shard

import (
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/internal/moreatomic"
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

	// Rescale is the function called, if Discord closes any of the gateways
	// with a 4011 close code aka. 'Sharding Required'.
	//
	// If the Manager was created using NewManager, this function will be set
	// to a function that automatically rescales the manager based on the
	// recommended number of shards, as received from Discord. If using any
	// other constructor, you need to provide a custom implementation for
	// this field, as otherwise all gateway connection will simply be closed.
	//
	// Keep in mind that if you are using a cache like the State does, you need
	// to wipe that cache before reconnecting to the gateway, as some cached
	// objects may be outdated when reconnecting.
	//
	// If you return nil or set this function to nil, all gateways will be
	// closed.
	Rescale     func() *Manager
	rescaleExec *moreatomic.Bool
}

// NewManager creates a Manager using as many gateways as recommended by
// Discord.
func NewManager(token string) (*Manager, error) {
	botData, err := gateway.BotURL(token)
	if err != nil {
		return nil, err
	}

	id := gateway.DefaultIdentifier(token)
	setStartLimiters(botData, id)

	m := newIdentifiedManager(gatewayURL(botData.URL), id, botData.Shards,
		GenerateShardIDs(botData.Shards)...)

	m.Rescale = func() *Manager {
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
//
// If you are using this constructor, you must provide a custom implementation
// for Manager.Rescale. Otherwise, if one of the gateway closes with
// a 'Sharding Required' error code, all other gateways will simply be closed.
func NewIdentifiedManager(
	id *gateway.Identifier, totalShards int, shardIDs ...int) (*Manager, error) {

	botData, err := gateway.BotURL(id.Token)
	if err != nil {
		return nil, err
	}

	setStartLimiters(botData, id)
	return newIdentifiedManager(gatewayURL(botData.URL), id, totalShards, shardIDs...), nil
}

func newIdentifiedManager(
	gatewayURL string, id *gateway.Identifier, totalShards int, shardIDs ...int) *Manager {

	gateways := make([]*gateway.Gateway, len(shardIDs))

	for i, shardID := range shardIDs {
		id.Shard = new(gateway.Shard)
		id.SetShard(shardID, totalShards)
		idCp := *id

		gateways[i] = gateway.NewCustomIdentifiedGateway(gatewayURL, &idCp)
	}

	return NewManagerWithGateways(gateways...)
}

// NewManagerWithShardIDs creates a new Manager using the passed token
// to create len(shardIDs) shards with the given ids.
//
// If you are using this constructor, you must provide a custom implementation
// for Manager.Rescale. Otherwise, if one of the gateway closes with a
// 'Sharding Required' error code, all other gateways will simply be closed.
func NewManagerWithShardIDs(token string, totalShards int, shardIDs ...int) (*Manager, error) {
	return NewIdentifiedManager(gateway.DefaultIdentifier(token), totalShards, shardIDs...)
}

// NewManagerWithGateways creates a new Manager from the given
// *gateways.gateways.
//
// If you are using this constructor, you must provide a custom implementation
// for Manager.Rescale. Otherwise, if one of the gateway closes with a
// 'Sharding Required' error code, all other gateways will simply be closed.
func NewManagerWithGateways(gateways ...*gateway.Gateway) *Manager {
	// user account will have a nil Shard, so check first
	numShards := 1
	if shard := gateways[0].Identifier.Shard; shard != nil {
		numShards = shard.NumShards()
	}

	m := &Manager{
		gateways:    gateways,
		mutex:       new(sync.RWMutex),
		Events:      make(chan interface{}),
		NumShards:   numShards,
		rescaleExec: new(moreatomic.Bool),
	}

	for _, g := range m.gateways {
		g.Events = m.Events
		g.OnShardingRequired = m.onShardingRequired
	}

	return m
}

// FromShardID returns the *gateway.Gateway with the given shard id, or nil if
// the shard manager has no gateways with the given id.
func (m *Manager) FromShardID(shardID int) *gateway.Gateway {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// fast-path, also prevent nil pointer dereference if this manager manages
	// a user account
	if m.NumShards == 1 {
		return m.gateways[0]
	}

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
func (m *Manager) Apply(f func(g *gateway.Gateway)) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, g := range m.gateways {
		f(g)
	}
}

// ApplyError is the same as Apply, but the iterator function returns an error.
// If such an error occurs, the error will be returned wrapped in an *Error.
//
// If all is set to true, ApplyError will apply the passed function to all
// gateways. If a single error occurs, it will be returned as an *Error, if
// multiple errors occur then they will be returned as *MultiError.
func (m *Manager) ApplyError(f func(g *gateway.Gateway) error, all bool) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var errs MultiError

	for _, g := range m.gateways {
		if err := f(g); err != nil {
			wrapperErr := &Error{
				ShardID: shardID(g),
				Source:  err,
			}

			if !all {
				return wrapperErr
			}

			errs = append(errs, wrapperErr)
		}
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errs
	}
}

// Gateways returns the gateways managed by this Manager.
func (m *Manager) Gateways() []*gateway.Gateway {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cp := make([]*gateway.Gateway, len(m.gateways))
	copy(cp, m.gateways)

	return cp
}

// AddIntents adds the passed gateway.Intents to all gateways managed by the
// Manager.
func (m *Manager) AddIntents(i gateway.Intents) {
	m.Apply(func(g *gateway.Gateway) {
		g.AddIntents(i)
	})
}

// Open opens all gateways handled by this Manager.
// If an error occurs, Open will attempt to close all previously opened
// gateways before returning.
func (m *Manager) Open() error {
	err := m.ApplyError(func(g *gateway.Gateway) error { return g.Open() }, false)
	if err == nil {
		return nil
	}

	var errs MultiError
	errs = append(errs, err)

	for shardID := 0; shardID < err.(*Error).ShardID; shardID++ {
		if shard := m.FromShardID(shardID); shard != nil { // exists?
			if err := shard.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errs
}

// Close closes all gateways handled by this Manager.
//
// If an error occurs, Close will attempt to close all remaining gateways
// first, before returning. If multiple errors occur during that process, a
// MultiError will be returned.
func (m *Manager) Close() error {
	return m.ApplyError(func(g *gateway.Gateway) error { return g.Close() }, true)
}

// Pause pauses all gateways managed by this Manager.
//
// If an error occurs, Pause will attempt to pause all remaining gateways
// first, before returning. If multiple errors occur during that process, a
// MultiError will be returned.
func (m *Manager) Pause() error {
	return m.ApplyError(func(g *gateway.Gateway) error { return g.Pause() }, true)
}

// UpdateStatus updates the status of all gateways handled by this Manager.
//
// If an error occurs, UpdateStatus will attempt to update the status of all
// remaining gateways first, before returning. If multiple errors occur during
// that process, a MultiError will be returned.
func (m *Manager) UpdateStatus(d gateway.UpdateStatusData) error {
	return m.ApplyError(func(g *gateway.Gateway) error { return g.UpdateStatus(d) }, true)
}

// RequestGuildMembers is used to request all members for a guild or a list of
// guilds. When initially connecting, if you don't have the GUILD_PRESENCES
// Gateway Intent, or if the guild is over 75k members, it will only send
// members who are in voice, plus the member for you (the connecting user).
// Otherwise, if a guild has over large_threshold members (value in the Gateway
// Identify), it will only send members who are online, have a role, have a
// nickname, or are in a voice channel, and if it has under large_threshold
// members, it will send all members. If a client wishes to receive additional
// members, they need to explicitly request them via this operation. The server
// will send Guild Members Chunk events in response with up to 1000 members per
// chunk until all members that match the request have been sent.
//
// Due to privacy and infrastructural concerns with this feature, there are
// some limitations that apply:
//
// 	1. GUILD_PRESENCES intent is required to set presences = true. Otherwise,
// 	   it will always be false
// 	2. GUILD_MEMBERS intent is required to request the entire member
// 	   list — (query=‘’, limit=0<=n)
// 	3. You will be limited to requesting 1 guild_id per request
// 	4. Requesting a prefix (query parameter) will return a maximum of 100
// 	   members
//
// Requesting user_ids will continue to be limited to returning 100 members
func (m *Manager) RequestGuildMembers(d gateway.RequestGuildMembersData) error {
	return m.FromGuildID(d.GuildIDs[0]).RequestGuildMembers(d)
}

// onShardingRequired is the function stored as Gateway.OnShardingRequired
// in every of the Manager's gateways.
func (m *Manager) onShardingRequired() {
	if m.rescaleExec.CompareAndSwap(false) {
		m.Close()

		if m.Rescale == nil {
			return
		}

		m.mutex.Lock()
		defer m.mutex.Unlock()

		newM := m.Rescale()
		if newM == nil {
			return
		}

		*m = *newM
	}
}

func shardID(g *gateway.Gateway) int {
	if shard := g.Identifier.Shard; shard != nil {
		return shard.ShardID()
	}

	return 0
}

func gatewayURL(baseURL string) string {
	param := url.Values{
		"v":        {gateway.Version},
		"encoding": {gateway.Encoding},
	}

	return baseURL + "?" + param.Encode()
}

func setStartLimiters(botData *gateway.BotData, id *gateway.Identifier) {
	resetAt := time.Now().Add(botData.StartLimit.ResetAfter.Duration())

	// Update the burst to be the current given time and reset it back to
	// the default when the given time is reached.
	id.IdentifyGlobalLimit.SetBurst(botData.StartLimit.Remaining)
	id.IdentifyGlobalLimit.SetBurstAt(resetAt, botData.StartLimit.Total)

	// Update the maximum number of identify requests allowed per 5s.
	id.IdentifyShortLimit.SetBurst(botData.StartLimit.MaxConcurrency)
}
