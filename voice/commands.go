package voice

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
)

// OPCode 0
// https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-identify-payload
type IdentifyData struct {
	GuildID   discord.Snowflake `json:"server_id"` // yes, this should be "server_id"
	UserID    discord.Snowflake `json:"user_id"`
	SessionID string            `json:"session_id"`
	Token     string            `json:"token"`
}

// Identify sends an Identify operation (opcode 0) to the Voice Gateway.
func (c *Connection) Identify() error {
	guildID := c.GuildID
	userID := c.UserID
	sessionID := c.SessionID
	token := c.Token

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

// SelectProtocol sends a Select Protocol operation (opcode 1) to the Voice Gateway.
func (c *Connection) SelectProtocol(data SelectProtocol) error {
	return c.Send(SelectProtocolOP, data)
}

// OPCode 3
// https://discordapp.com/developers/docs/topics/voice-connections#heartbeating-example-heartbeat-payload
type Heartbeat uint64

// Heartbeat sends a Heartbeat operation (opcode 3) to the Voice Gateway.
func (c *Connection) Heartbeat() error {
	return c.Send(HeartbeatOP, time.Now().UnixNano())
}

// https://discordapp.com/developers/docs/topics/voice-connections#speaking
type Speaking uint64

const (
	Microphone Speaking = 1 << iota
	Soundshare
	Priority
)

// OPCode 5
// https://discordapp.com/developers/docs/topics/voice-connections#speaking-example-speaking-payload
type SpeakingData struct {
	Speaking Speaking `json:"speaking"`
	Delay    int      `json:"delay"`
	SSRC     uint32   `json:"ssrc"`
}

// Speaking sends a Speaking operation (opcode 5) to the Voice Gateway.
func (c *Connection) Speaking(s Speaking) error {
	// How do we allow a user to stop speaking?
	// Also: https://discordapp.com/developers/docs/topics/voice-connections#voice-data-interpolation

	return c.Send(SpeakingOP, SpeakingData{
		Speaking: s,
		Delay:    0,
		SSRC:     c.ready.SSRC,
	})
}

// StopSpeaking stops speaking.
// https://discordapp.com/developers/docs/topics/voice-connections#voice-data-interpolation
func (c *Connection) StopSpeaking() {
	for i := 0; i < 5; i++ {
		c.OpusSend <- []byte{0xF8, 0xFF, 0xFE}
	}
}

// OPCode 7
// https://discordapp.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resume-connection-payload
type ResumeData struct {
	GuildID   discord.Snowflake `json:"server_id"` // yes, this should be "server_id"
	SessionID string            `json:"session_id"`
	Token     string            `json:"token"`
}

// Resume sends a Resume operation (opcode 7) to the Voice Gateway.
func (c *Connection) Resume() error {
	guildID := c.GuildID
	sessionID := c.SessionID
	token := c.Token

	if guildID == 0 || sessionID == "" || token == "" {
		return ErrMissingForResume
	}

	return c.Send(ResumeOP, ResumeData{
		GuildID:   guildID,
		SessionID: sessionID,
		Token:     token,
	})
}
