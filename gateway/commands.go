package gateway

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/pkg/errors"
)

// Rules: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent

// Identify structure is at identify.go

func (i *IdentifyData) SetShard(id, num int) {
	if i.Shard == nil {
		i.Shard = new(Shard)
	}
	i.Shard[0], i.Shard[1] = id, num
}

func (g *Gateway) Identify() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	if err := g.Identifier.Wait(ctx); err != nil {
		return errors.Wrap(err, "Can't wait for identify()")
	}

	return g.send(false, IdentifyOP, g.Identifier)
}

type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// Resume sends to the Websocket a Resume OP, but it doesn't actually resume
// from a dead connection. Start() resumes from a dead connection.
func (g *Gateway) Resume() error {
	var (
		ses = g.SessionID
		seq = g.Sequence.Get()
	)

	if ses == "" || seq == 0 {
		return ErrMissingForResume
	}

	return g.send(false, ResumeOP, ResumeData{
		Token:     g.Identifier.Token,
		SessionID: ses,
		Sequence:  seq,
	})
}

// HeartbeatData is the last sequence number to be sent.
type HeartbeatData int

func (g *Gateway) Heartbeat() error {
	// g.available.RLock()
	// defer g.available.RUnlock()
	return g.Send(HeartbeatOP, g.Sequence.Get())
}

type RequestGuildMembersData struct {
	GuildID []discord.Snowflake `json:"guild_id"`
	UserIDs []discord.Snowflake `json:"user_ids,omitempty"`

	Query     string `json:"query,omitempty"`
	Limit     uint   `json:"limit"`
	Presences bool   `json:"presences,omitempty"`
}

func (g *Gateway) RequestGuildMembers(data RequestGuildMembersData) error {
	return g.Send(RequestGuildMembersOP, data)
}

type UpdateVoiceStateData struct {
	GuildID   discord.Snowflake `json:"guild_id"`
	ChannelID discord.Snowflake `json:"channel_id"`
	SelfMute  bool              `json:"self_mute"`
	SelfDeaf  bool              `json:"self_deaf"`
}

func (g *Gateway) UpdateVoiceState(data UpdateVoiceStateData) error {
	return g.Send(VoiceStateUpdateOP, data)
}

type UpdateStatusData struct {
	Since discord.UnixMsTimestamp `json:"since"` // 0 if not idle

	Game       *discord.Activity   `json:"game,omitempty"` // nullable
	Activities *[]discord.Activity `json:"activities,omitempty"`

	Status discord.Status `json:"status"`
	AFK    bool           `json:"afk"`
}

func (g *Gateway) UpdateStatus(data UpdateStatusData) error {
	return g.Send(StatusUpdateOP, data)
}

// Undocumented
type GuildSubscribeData struct {
	Typing     bool              `json:"typing"`
	Activities bool              `json:"activities"`
	GuildID    discord.Snowflake `json:"guild_id"`

	// Channels is not documented. It's used to fetch the right members sidebar.
	Channels map[discord.Snowflake][][2]int `json:"channels"`
}

func (g *Gateway) GuildSubscribe(data GuildSubscribeData) error {
	return g.Send(GuildSubscriptionsOP, data)
}
