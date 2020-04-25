package voicegateway

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/pkg/errors"
)

var (
	// ErrMissingForIdentify is an error when we are missing information to identify.
	ErrMissingForIdentify = errors.New("missing GuildID, UserID, SessionID, or Token for identify")

	// ErrMissingForResume is an error when we are missing information to resume.
	ErrMissingForResume = errors.New("missing GuildID, SessionID, or Token for resuming")
)

// OPCode 0
// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-identify-payload
type IdentifyData struct {
	GuildID   discord.Snowflake `json:"server_id"` // yes, this should be "server_id"
	UserID    discord.Snowflake `json:"user_id"`
	SessionID string            `json:"session_id"`
	Token     string            `json:"token"`
}

// Identify sends an Identify operation (opcode 0) to the Gateway Gateway.
func (c *Gateway) Identify() error {
	guildID := c.state.GuildID
	userID := c.state.UserID
	sessionID := c.state.SessionID
	token := c.state.Token

	if guildID == 0 || userID == 0 || sessionID == "" || token == "" {
		return ErrMissingForIdentify
	}

	return c.Send(IdentifyOP, IdentifyData{
		GuildID:   guildID,
		UserID:    userID,
		SessionID: sessionID,
		Token:     token,
	})
}

// OPCode 1
// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-udp-connection-example-select-protocol-payload
type SelectProtocol struct {
	Protocol string             `json:"protocol"`
	Data     SelectProtocolData `json:"data"`
}

type SelectProtocolData struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Mode    string `json:"mode"`
}

// SelectProtocol sends a Select Protocol operation (opcode 1) to the Gateway Gateway.
func (c *Gateway) SelectProtocol(data SelectProtocol) error {
	return c.Send(SelectProtocolOP, data)
}

// OPCode 3
// https://discordapp.com/developers/docs/topics/voice-connections#heartbeating-example-heartbeat-payload
type Heartbeat uint64

// Heartbeat sends a Heartbeat operation (opcode 3) to the Gateway Gateway.
func (c *Gateway) Heartbeat() error {
	return c.Send(HeartbeatOP, time.Now().UnixNano())
}

// https://discordapp.com/developers/docs/topics/voice-connections#speaking
type SpeakingFlag uint64

const (
	Microphone SpeakingFlag = 1 << iota
	Soundshare
	Priority
)

// OPCode 5
// https://discordapp.com/developers/docs/topics/voice-connections#speaking-example-speaking-payload
type SpeakingData struct {
	Speaking SpeakingFlag `json:"speaking"`
	Delay    int          `json:"delay"`
	SSRC     uint32       `json:"ssrc"`
}

// Speaking sends a Speaking operation (opcode 5) to the Gateway Gateway.
func (c *Gateway) Speaking(flag SpeakingFlag) error {
	// How do we allow a user to stop speaking?
	// Also: https://discordapp.com/developers/docs/topics/voice-connections#voice-data-interpolation

	return c.Send(SpeakingOP, SpeakingData{
		Speaking: flag,
		Delay:    0,
		SSRC:     c.ready.SSRC,
	})
}

// OPCode 7
// https://discordapp.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resume-connection-payload
type ResumeData struct {
	GuildID   discord.Snowflake `json:"server_id"` // yes, this should be "server_id"
	SessionID string            `json:"session_id"`
	Token     string            `json:"token"`
}

// Resume sends a Resume operation (opcode 7) to the Gateway Gateway.
func (c *Gateway) Resume() error {
	guildID := c.state.GuildID
	sessionID := c.state.SessionID
	token := c.state.Token

	if !guildID.Valid() || sessionID == "" || token == "" {
		return ErrMissingForResume
	}

	return c.Send(ResumeOP, ResumeData{
		GuildID:   guildID,
		SessionID: sessionID,
		Token:     token,
	})
}
