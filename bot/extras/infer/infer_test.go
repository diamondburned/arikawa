package infer

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
)

type hasID struct {
	ChannelID discord.ChannelID
}

type embedsID struct {
	*hasID
	*embedsID
}

type hasChannelInName struct {
	ID discord.ChannelID
}

func TestReflectChannelID(t *testing.T) {
	var s = &hasID{
		ChannelID: 69420,
	}

	t.Run("hasID", func(t *testing.T) {
		if id := ChannelID(s); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("embedsID", func(t *testing.T) {
		var e = &embedsID{
			hasID: s,
		}

		if id := ChannelID(e); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})

	t.Run("hasChannelInName", func(t *testing.T) {
		var s = &hasChannelInName{
			ID: 69420,
		}

		if id := ChannelID(s); id != 69420 {
			t.Fatal("unexpected channelID:", id)
		}
	})
}

func BenchmarkReflectChannelID_1Level(b *testing.B) {
	var s = &hasID{
		ChannelID: 69420,
	}

	for i := 0; i < b.N; i++ {
		if id := ChannelID(s); id != s.ChannelID {
			b.Fatal("Unexpected ChannelID:", id)
		}
	}
}

func BenchmarkReflectChannelID_5Level(b *testing.B) {
	var s = &embedsID{
		nil,
		&embedsID{
			nil,
			&embedsID{
				nil,
				&embedsID{
					hasID: &hasID{
						ChannelID: 69420,
					},
				},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		if id := ChannelID(s); id != 69420 {
			b.Fatal("Unexpected ChannelID:", id)
		}
	}
}
