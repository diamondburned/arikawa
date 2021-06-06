package voicegateway

import (
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
)

// OPCode 2
// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-ready-payload
type ReadyEvent struct {
	IP          string   `json:"ip"`
	Modes       []string `json:"modes"`
	Experiments []string `json:"experiments"`
	Port        int      `json:"port"`
	SSRC        uint32   `json:"ssrc"`

	// From Discord's API Docs:
	//
	// `heartbeat_interval` here is an erroneous field and should be ignored.
	// The correct `heartbeat_interval` value comes from the Hello payload.

	// HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
}

func (r ReadyEvent) Addr() string {
	return r.IP + ":" + strconv.Itoa(r.Port)
}

// OPCode 4
// https://discord.com/developers/docs/topics/voice-connections#establishing-a-voice-udp-connection-example-session-description-payload
type SessionDescriptionEvent struct {
	Mode      string   `json:"mode"`
	SecretKey [32]byte `json:"secret_key"`
}

// OPCode 5
type SpeakingEvent SpeakingData

// OPCode 6
// https://discord.com/developers/docs/topics/voice-connections#heartbeating-example-heartbeat-ack-payload
type HeartbeatACKEvent uint64

// OPCode 8
// https://discord.com/developers/docs/topics/voice-connections#heartbeating-example-hello-payload-since-v3
type HelloEvent struct {
	HeartbeatInterval discord.Milliseconds `json:"heartbeat_interval"`
}

// OPCode 9
// https://discord.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resumed-payload
type ResumedEvent struct{}

// OPCode 12
// (undocumented)
type ClientConnectEvent struct {
	UserID    discord.UserID `json:"user_id"`
	AudioSSRC uint32         `json:"audio_ssrc"`
	VideoSSRC uint32         `json:"video_ssrc"`
}

// OPCode 13
// Undocumented, existence mentioned in below issue
// https://github.com/discord/discord-api-docs/issues/510
type ClientDisconnectEvent struct {
	UserID discord.UserID `json:"user_id"`
}
