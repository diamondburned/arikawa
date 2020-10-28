package gateway

import (
	"context"

	"github.com/diamondburned/arikawa/discord"
	"github.com/pkg/errors"
)

// Rules: VOICE_STATE_UPDATE -> VoiceStateUpdateEvent

// Identify structure is at identify.go

func (g *Gateway) Identify() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.IdentifyCtx(ctx)
}

func (g *Gateway) IdentifyCtx(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, g.WSTimeout)
	defer cancel()

	if err := g.Identifier.Wait(ctx); err != nil {
		return errors.Wrap(err, "can't wait for identify()")
	}

	return g.SendCtx(ctx, IdentifyOP, g.Identifier)
}

type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

// Resume sends to the Websocket a Resume OP, but it doesn't actually resume
// from a dead connection. Start() resumes from a dead connection.
func (g *Gateway) Resume() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.ResumeCtx(ctx)
}

// ResumeCtx sends to the Websocket a Resume OP, but it doesn't actually resume
// from a dead connection. Start() resumes from a dead connection.
func (g *Gateway) ResumeCtx(ctx context.Context) error {
	var (
		ses = g.SessionID
		seq = g.Sequence.Get()
	)

	if ses == "" || seq == 0 {
		return ErrMissingForResume
	}

	return g.SendCtx(ctx, ResumeOP, ResumeData{
		Token:     g.Identifier.Token,
		SessionID: ses,
		Sequence:  seq,
	})
}

// HeartbeatData is the last sequence number to be sent.
type HeartbeatData int

func (g *Gateway) Heartbeat() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.HeartbeatCtx(ctx)
}

func (g *Gateway) HeartbeatCtx(ctx context.Context) error {
	return g.SendCtx(ctx, HeartbeatOP, g.Sequence.Get())
}

type RequestGuildMembersData struct {
	GuildID []discord.GuildID `json:"guild_id"`
	UserIDs []discord.UserID  `json:"user_ids,omitempty"`

	Query     string `json:"query"`
	Limit     uint   `json:"limit"`
	Presences bool   `json:"presences,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

func (g *Gateway) RequestGuildMembers(data RequestGuildMembersData) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.RequestGuildMembersCtx(ctx, data)
}

func (g *Gateway) RequestGuildMembersCtx(
	ctx context.Context, data RequestGuildMembersData) error {

	return g.SendCtx(ctx, RequestGuildMembersOP, data)
}

type UpdateVoiceStateData struct {
	GuildID   discord.GuildID   `json:"guild_id"`
	ChannelID discord.ChannelID `json:"channel_id"` // nullable
	SelfMute  bool              `json:"self_mute"`
	SelfDeaf  bool              `json:"self_deaf"`
}

func (g *Gateway) UpdateVoiceState(data UpdateVoiceStateData) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.UpdateVoiceStateCtx(ctx, data)
}

func (g *Gateway) UpdateVoiceStateCtx(
	ctx context.Context, data UpdateVoiceStateData) error {

	return g.SendCtx(ctx, VoiceStateUpdateOP, data)
}

type UpdateStatusData struct {
	Since discord.UnixMsTimestamp `json:"since"` // 0 if not idle

	// Both fields are nullable.
	Activities *[]discord.Activity `json:"activities,omitempty"`

	Status discord.Status `json:"status"`
	AFK    bool           `json:"afk"`
}

func (g *Gateway) UpdateStatus(data UpdateStatusData) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.UpdateStatusCtx(ctx, data)
}

func (g *Gateway) UpdateStatusCtx(ctx context.Context, data UpdateStatusData) error {
	return g.SendCtx(ctx, StatusUpdateOP, data)
}

// Undocumented
type GuildSubscribeData struct {
	Typing     bool            `json:"typing"`
	Activities bool            `json:"activities"`
	GuildID    discord.GuildID `json:"guild_id"`

	// Channels is not documented. It's used to fetch the right members sidebar.
	Channels map[discord.ChannelID][][2]int `json:"channels,omitempty"`
}

func (g *Gateway) GuildSubscribe(data GuildSubscribeData) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
	defer cancel()

	return g.GuildSubscribeCtx(ctx, data)
}

func (g *Gateway) GuildSubscribeCtx(ctx context.Context, data GuildSubscribeData) error {
	return g.SendCtx(ctx, GuildSubscriptionsOP, data)
}
