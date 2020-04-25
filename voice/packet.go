package voice

// https://discordapp.com/developers/docs/topics/voice-connections#encrypting-and-sending-voice
type Packet struct {
	Version   byte   // Single byte value of 0x80 - 1 byte
	Type      byte   // Single byte value of 0x78 - 1 byte
	Sequence  uint16 // Unsigned short (big endian) - 4 bytes
	Timestamp uint32 // Unsigned integer (big endian) - 4 bytes
	SSRC      uint32 // Unsigned integer (big endian) - 4 bytes
	Opus      []byte // Binary data
}
